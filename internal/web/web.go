// Package web serves taskman's read-only web UI: a single page listing every
// task, kept live over datastar/SSE. Unlike the CLI commands, it is a
// long-running server, not a one-shot task transition.
package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/mhmdkzr/taskman/internal/commands/list"
	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/web/components"
)

// pollInterval is how often an open SSE stream re-reads the store to detect a
// change worth pushing to the browser.
const pollInterval = time.Second

// shutdownTimeout bounds how long Run waits for in-flight requests to finish.
const shutdownTimeout = 5 * time.Second

// Server serves the read-only web UI from a task store.
type Server struct {
	store *store.Store
}

// NewServer returns a Server reading tasks from st.
func NewServer(st *store.Store) *Server {
	return &Server{store: st}
}

// Handler returns the web UI's routes: the page and its datastar SSE stream.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /tasks", s.handleTasks)
	return mux
}

// Run serves the web UI on addr until ctx is canceled, then shuts the server
// down gracefully. A normal shutdown returns nil.
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: shutdownTimeout,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	slog.Info("web ui listening", "addr", addr)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown web server: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve web: %w", err)
	}
}

// handleIndex renders the whole page, task list included.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tasks, err := list.List(r.Context(), s.store)
	if err != nil {
		slog.Error("list tasks", "error", err)
		http.Error(w, "list tasks", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := components.Page(tasks).Render(r.Context(), w); err != nil {
		// The response has already begun; there is nothing left to send the
		// client. The failure still must not be silent.
		slog.Error("render page", "error", err)
	}
}

// handleTasks is the datastar SSE endpoint. It long-polls the store and
// patches the task list into the page whenever the stored tasks change.
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	last, err := s.renderTaskList(r.Context())
	if err != nil {
		slog.Error("list tasks", "error", err)
		http.Error(w, "list tasks", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			current, err := s.renderTaskList(r.Context())
			if err != nil {
				slog.Error("list tasks", "error", err)
				return
			}
			if current == last {
				continue
			}
			last = current
			err = sse.PatchElements(
				current,
				datastar.WithSelectorID(components.TasksID),
				datastar.WithModeInner(),
			)
			if err != nil {
				slog.Error("patch tasks", "error", err)
				return
			}
		}
	}
}

// renderTaskList reads every task and renders just the list fragment, so it
// can be compared against the previous render and patched into the page.
func (s *Server) renderTaskList(ctx context.Context) (string, error) {
	tasks, err := list.List(ctx, s.store)
	if err != nil {
		return "", fmt.Errorf("list tasks: %w", err)
	}
	var b strings.Builder
	if err := components.TaskList(tasks).Render(ctx, &b); err != nil {
		return "", fmt.Errorf("render task list: %w", err)
	}
	return b.String(), nil
}
