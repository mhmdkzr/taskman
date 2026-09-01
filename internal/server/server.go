// Package server implements the long-running agent server: it owns the agent
// runtime (store, shared tool set, scheduler, consumer) and answers client
// commands arriving as NATS core request/reply on the protocol subject.
// Requests are handled one at a time: the handler runs synchronously in the
// subscription callback, so NATS delivers and serializes them.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/mhmdkzr/taskman/internal/agent"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/protocol"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/scheduler"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/tools/spawn"
	"github.com/nats-io/nats.go"
	"github.com/zendev-sh/goai"
)

// drainTimeout bounds how long shutdown waits for in-flight runs and
// sub-agents before giving up.
const drainTimeout = 30 * time.Second

// Server is the long-running agent runtime. Build it with New, Run it until
// the context is cancelled, and release the store with Close.
type Server struct {
	log      *log.Logger
	pub      publisher.Publisher
	st       *store.Store
	opts     agent.Options
	sched    *scheduler.Scheduler
	tools    []goai.Tool
	runner   *spawn.Runner
	consumer *agent.Consumer
	started  time.Time
}

// New assembles the server runtime: the store, the durable scheduler, the
// shared tool set (and sub-agent runner), and the scheduled-run consumer. opts
// carries the provider config and defaults; pub carries the NATS connection
// the server replies on (pub.Conn()).
func New(opts agent.Options, pub publisher.Publisher) (*Server, error) {
	if pub.Conn() == nil {
		return nil, errors.New("server: publisher has no NATS connection")
	}
	if opts.Config.DBPath == "" {
		opts.Config.DBPath = config.DefaultDBPath
	}
	dbPath, err := config.ExpandHome(opts.Config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("server: db path: %w", err)
	}
	opts.Config.DBPath = dbPath

	st, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("server: open store: %w", err)
	}
	if err := store.Migrate(st.RW()); err != nil {
		st.Close()
		return nil, fmt.Errorf("server: migrate: %w", err)
	}

	sched := scheduler.New(st.RW(), pub, 0)
	tools, runner, err := agent.DefaultTools(st, pub, opts)
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("server: build tools: %w", err)
	}
	consumer := agent.NewConsumerWithTools(st, opts, pub, 1, tools, runner)

	return &Server{
		pub:      pub,
		st:       st,
		opts:     opts,
		sched:    sched,
		tools:    tools,
		runner:   runner,
		consumer: consumer,
		started:  time.Now(),
	}, nil
}

// Close releases the store. It does not close the NATS connection; the caller
// owns that.
func (s *Server) Close() error {
	return s.st.Close()
}

// Run serves the server until ctx is cancelled: it starts the scheduler and
// consumer, subscribes to the request subject, and dispatches each request in
// the subscription callback. Because the handler blocks until a run completes,
// NATS delivers requests one at a time and they are served sequentially.
func (s *Server) Run(ctx context.Context, logger *log.Logger) error {
	s.log = logger
	if s.log == nil {
		s.log = log.New(io.Discard, "", 0)
	}

	errc := make(chan error, 2)
	go func() { errc <- s.sched.Run(ctx) }()
	go func() { errc <- s.consumer.Run(ctx) }()

	sub, err := s.pub.Conn().Subscribe(protocol.Subject, s.handle)
	if err != nil {
		return fmt.Errorf("server: subscribe %s: %w", protocol.Subject, err)
	}
	defer sub.Unsubscribe()
	if err := s.pub.Conn().Flush(); err != nil {
		return fmt.Errorf("server: flush: %w", err)
	}
	s.log.Printf("agent server serving on %s", protocol.Subject)

	select {
	case err := <-errc:
		if err != nil {
			return fmt.Errorf("server runtime: %w", err)
		}
	case <-ctx.Done():
	}
	s.log.Printf("shutting down; draining in-flight runs")
	drainCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()
	if err := s.consumer.Wait(drainCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		s.log.Printf("drain: %v", err)
	}
	return nil
}

// handle dispatches one request envelope. It runs synchronously so requests
// are served one at a time.
func (s *Server) handle(msg *nats.Msg) {
	var req protocol.Request
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		s.respond(msg, protocol.Reply{Error: fmt.Sprintf("decode request: %v", err)})
		return
	}
	switch req.Command {
	case protocol.CommandRun:
		s.handleRun(msg, req.Data)
	case protocol.CommandPing:
		s.handlePing(msg)
	default:
		s.respond(msg, protocol.Reply{Command: req.Command, Error: fmt.Sprintf("unknown command %q", req.Command)})
	}
}

func (s *Server) handleRun(msg *nats.Msg, data json.RawMessage) {
	reply := protocol.Reply{Command: protocol.CommandRun}
	var req protocol.RunRequest
	if err := json.Unmarshal(data, &req); err != nil {
		reply.Error = fmt.Sprintf("decode run request: %v", err)
		s.respond(msg, reply)
		return
	}
	if err := req.Validate(); err != nil {
		reply.Error = err.Error()
		s.respond(msg, reply)
		return
	}

	ctx := context.Background()
	opts := s.resolve(req)

	var (
		res       *agent.Result
		sessionID string
		err       error
	)
	switch {
	case req.ForkFrom != "":
		res, sessionID, err = agent.ForkRun(ctx, s.st, opts, s.pub, req.ForkFrom, req.ForkTurn, req.Prompt)
	case req.SessionID != "":
		res, sessionID, err = agent.ContinueSession(ctx, s.st, opts, s.pub, req.SessionID, req.Prompt)
	default:
		res, err = agent.Run(ctx, opts, s.pub, req.Prompt, "")
		if err == nil {
			sessionID, err = agent.PersistRun(ctx, s.st, opts, s.pub, req.Prompt, res)
		}
	}
	if err != nil {
		reply.Error = err.Error()
		s.respond(msg, reply)
		return
	}

	runReply := protocol.RunReply{SessionID: sessionID, Text: res.Text}
	if req.AsJSON {
		out := agent.NewRunOutput(sessionID, opts, res.Usage, res.Messages, res.Steps)
		if runReply.RunOutput, err = json.MarshalIndent(out, "", "  "); err != nil {
			reply.Error = err.Error()
			s.respond(msg, reply)
			return
		}
	}
	reply.OK = true
	if reply.Data, err = json.Marshal(runReply); err != nil {
		reply.OK = false
		reply.Error = err.Error()
		reply.Data = nil
	}
	s.respond(msg, reply)
}

func (s *Server) handlePing(msg *nats.Msg) {
	reply := protocol.Reply{Command: protocol.CommandPing, OK: true}
	ping := protocol.PingReply{
		Version: protocol.Version,
		Uptime:  time.Since(s.started).Round(time.Second).String(),
	}
	reply.Data, _ = json.Marshal(ping)
	s.respond(msg, reply)
}

// resolve overlays a run request's options onto the server's defaults, keeping
// the shared tool set so runs inherit the full default tooling. Empty and
// sentinel ("default") override fields are ignored.
func (s *Server) resolve(req protocol.RunRequest) agent.Options {
	o := s.opts
	o.Tools = s.tools
	if m := agent.MeaningfulOverride(req.Model); m != "" {
		o.Model = m
	}
	if e := agent.MeaningfulOverride(req.ReasoningEffort); e != "" {
		o.ReasoningEffort = agent.ReasoningEffort(e)
	}
	if req.MaxSteps > 0 {
		o.MaxSteps = req.MaxSteps
	}
	if sp := agent.MeaningfulOverride(req.SystemPrompt); sp != "" {
		o.SystemPrompt = sp
	}
	return o
}

// respond sends a reply envelope back to the request's inbox. Errors are
// logged and best-effort.
func (s *Server) respond(msg *nats.Msg, reply protocol.Reply) {
	data, err := json.Marshal(reply)
	if err != nil {
		s.log.Printf("marshal reply: %v", err)
		_ = msg.Respond([]byte(`{"ok":false,"error":"internal error"}`))
		return
	}
	if err := msg.Respond(data); err != nil {
		s.log.Printf("respond to %s: %v", msg.Subject, err)
	}
}
