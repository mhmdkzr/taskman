package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/tools/spawn"
	"github.com/zendev-sh/goai"
)

// consumerName is the durable JetStream consumer bound to the agent.run
// subject. Being durable, a restart resumes from where the previous process
// left off instead of re-reading the whole stream.
const consumerName = "agent-run"

// Consumer runs scheduled agent runs. It consumes the agent.run subject as a
// durable JetStream consumer on the TASKMAN stream. The scheduler publishes each
// message with its schedule id as the dedup id, so a re-publish (e.g. a
// replay after a crash between publish and marking it) is dropped by the
// stream and delivery is exactly-once. A message is acknowledged only once its
// run completes; a crash mid-run leaves it unacked, so it is redelivered on
// restart rather than lost. A Consumer is not safe for concurrent use beyond
// the intended Run/Wait lifecycle.
type Consumer struct {
	st      *store.Store
	base    Options
	pub     publisher.Publisher
	workers int
	sem     chan struct{}

	tools  []goai.Tool
	runner *spawn.Runner

	ready chan struct{}

	inflight sync.WaitGroup
}

// NewConsumer builds a Consumer. base carries the defaults scheduled runs
// inherit (model, effort, max steps, system prompt, config); pub is the event
// bus; workers bounds concurrent runs (non-positive means 1). The shared tool
// set is built once here and reused by every run.
func NewConsumer(st *store.Store, base Options, pub publisher.Publisher, workers int) (*Consumer, error) {
	tools, runner, err := DefaultTools(st, pub, base)
	if err != nil {
		return nil, fmt.Errorf("consumer: build tools: %w", err)
	}
	return NewConsumerWithTools(st, base, pub, workers, tools, runner), nil
}

// NewConsumerWithTools builds a Consumer around an already-built tool set and
// sub-agent Runner. It exists so the server can build the tools once and share
// them between the request handler and the consumer: two tool sets would mean
// two sub-agent registries. The caller owns tools and runner.
func NewConsumerWithTools(st *store.Store, base Options, pub publisher.Publisher, workers int, tools []goai.Tool, runner *spawn.Runner) *Consumer {
	if workers <= 0 {
		workers = 1
	}
	return &Consumer{
		st:      st,
		base:    base,
		pub:     pub,
		workers: workers,
		sem:     make(chan struct{}, workers),
		tools:   tools,
		runner:  runner,
		ready:   make(chan struct{}),
	}
}

// Ready returns a channel closed once the consumer is subscribed to agent.run
// and the subscription is live, so publishers can avoid dropping messages on a
// connection that has not finished registering yet.
func (c *Consumer) Ready() <-chan struct{} {
	return c.ready
}

// Run serves the consumer until ctx is cancelled: it subscribes to agent.run
// on the TASKMAN stream, then blocks. On cancellation it stops the subscription,
// waits for in-flight runs to finish, and returns nil.
func (c *Consumer) Run(ctx context.Context) error {
	sub, err := c.pub.Subscribe(ctx, RunSubject, consumerName, func(msg publisher.Message) {
		go c.process(msg)
	})
	if err != nil {
		return fmt.Errorf("consumer: subscribe %s: %w", RunSubject, err)
	}
	close(c.ready)
	<-ctx.Done()
	sub.Stop()
	c.inflight.Wait()
	return nil
}

// Wait blocks until every run started by the consumer and the sub-agents they
// spawned have finished, and returns nil (or ctx.Err() if cancelled first).
// Call it before the process exits so scheduled work is not cut off mid-run.
func (c *Consumer) Wait(ctx context.Context) error {
	c.inflight.Wait()
	return c.runner.Wait(ctx)
}

// process runs one agent.run message. Runs complete even after the consumer is
// stopped: a message already taken off the subject must not be lost, so it is
// acknowledged only once the run has finished.
func (c *Consumer) process(msg publisher.Message) {
	c.sem <- struct{}{}
	defer func() { <-c.sem }()
	c.inflight.Add(1)
	defer c.inflight.Done()

	ctx := context.Background()
	var req RunRequest
	if err := json.Unmarshal(msg.Data(), &req); err != nil {
		c.fail(ctx, req, fmt.Errorf("consumer: unmarshal %s message: %w", RunSubject, err))
		msg.Ack()
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		c.fail(ctx, req, fmt.Errorf("consumer: %s message has no prompt", RunSubject))
		msg.Ack()
		return
	}

	opts := c.resolve(req)

	// The session id is known before the run starts: continue uses the
	// request's session, a fresh run pre-generates its id so every event of
	// the run (including AgentRunStarted) carries it.
	runSessionID := req.SessionID
	if runSessionID == "" {
		var genErr error
		runSessionID, genErr = store.NewSessionID()
		if genErr != nil {
			c.fail(ctx, req, fmt.Errorf("consumer: generate session id: %w", genErr))
			msg.Ack()
			return
		}
	}
	c.publish(ctx, events.AgentRunStarted{
		SessionID: runSessionID,
		Prompt:    req.Prompt,
		Model:     opts.Model,
	})

	var (
		res *Result
		err error
	)
	if req.SessionID != "" {
		res, _, err = ContinueSession(ctx, c.st, opts, c.pub, req.SessionID, req.Prompt)
	} else {
		res, err = Run(ctx, opts, c.pub, req.Prompt, runSessionID)
		if err == nil {
			_, err = PersistRun(ctx, c.st, opts, c.pub, req.Prompt, res)
		}
	}
	if err != nil {
		c.fail(ctx, req, err)
		msg.Ack()
		return
	}
	c.publish(ctx, events.AgentRunFinished{
		SessionID: runSessionID,
		Prompt:    req.Prompt,
		Text:      res.Text,
		Usage:     usageToEvent(res.Usage),
		Steps:     len(res.Steps),
	})
	msg.Ack()
}

// publish sends an event on the bus with a bounded wait so a stalled bus
// cannot hang a scheduled run. Publish errors are best-effort and ignored.
func (c *Consumer) publish(ctx context.Context, e publisher.Event[any]) {
	pubCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	_ = c.pub.Publish(pubCtx, e)
}

func (c *Consumer) fail(ctx context.Context, req RunRequest, err error) {
	c.publish(ctx, events.AgentRunFailed{
		SessionID: req.SessionID,
		Prompt:    req.Prompt,
		Error:     err.Error(),
	})
}

// resolve overlays the request's options onto the consumer's defaults. The
// shared tool set is always used so scheduled runs inherit the full default
// tooling, including the schedule_agent tool. Empty and sentinel ("default")
// override fields are ignored, so an LLM that fills them with placeholders
// does not break the run.
func (c *Consumer) resolve(req RunRequest) Options {
	o := c.base
	o.Tools = c.tools
	if m := MeaningfulOverride(req.Model); m != "" {
		o.Model = m
	}
	if e := MeaningfulOverride(req.ReasoningEffort); e != "" {
		o.ReasoningEffort = ReasoningEffort(e)
	}
	if req.MaxSteps > 0 {
		o.MaxSteps = req.MaxSteps
	}
	if sp := MeaningfulOverride(req.SystemPrompt); sp != "" {
		o.SystemPrompt = sp
	}
	return o
}
