// Package web serves a read-only, realtime dashboard for the task backlog
// and the agent bus: it renders tasks and sessions from the SQLite store and
// streams live agent/pipeline events over Server-Sent Events. It is the
// browser's only view of the system — everything it shows comes from the
// store or the bus, never from the browser. The one write the browser may
// make is to POST commands (e.g. run these tasks), which the server forwards
// as core NATS command messages for a runner to consume.
package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mhmdkzr/taskman/internal/pipeline"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
)

// Config carries what the web server needs: a store for read-only queries
// (Store.RO) and the pipeline runner configuration used to execute commands.
type Config struct {
	// Addr is the listen address, e.g. "127.0.0.1:8080".
	Addr string

	// Store backs every read API. RO queries run through Store.RO; the
	// command runner writes through Store.RW via Pipeline.Store.
	Store *store.Store

	// Pub is the NATS gateway: SSE streams subscribe live on it, and
	// commands are published through it.
	Pub publisher.Publisher

	// Pipeline is the runner configuration command execution uses. It is
	// shared across every task the web server runs.
	Pipeline pipeline.Config
}

// Server is the web dashboard. Create with New, then serve it with Run.
type Server struct {
	cfg Config
}

// New validates cfg and returns a Server ready to Run.
func New(cfg Config) (*Server, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("web: addr is required")
	}
	if cfg.Store == nil {
		return nil, fmt.Errorf("web: store is required")
	}
	if cfg.Pub.Conn() == nil {
		return nil, fmt.Errorf("web: publisher has no NATS connection")
	}
	cfg.Pipeline.Store = cfg.Store
	return &Server{cfg: cfg}, nil
}

// Handler returns the server's HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /events", s.handleEvents)
	mux.HandleFunc("GET /api/overview", s.handleOverview)
	mux.HandleFunc("POST /api/commands", s.handleCommands)
	return mux
}

// Run serves the dashboard and executes commands until ctx is cancelled or
// the server fails. It blocks.
func (s *Server) Run(ctx context.Context) error {
	slog.Info("web: dashboard listening", "addr", s.cfg.Addr)

	stopCommands, err := s.serveCommands(ctx)
	if err != nil {
		return err
	}
	defer stopCommands()

	httpServer := &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.ListenAndServe() }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown web server: %w", err)
		}
		return nil
	}
}
