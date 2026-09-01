// Package web serves a read-only, realtime dashboard for the task backlog and
// the agent bus, built with Datastar: the server renders HTML fragments and
// pushes them into the DOM over SSE as bus events arrive. It is purely
// observational — there is no control surface here. Controls live in the
// taskman CLI, which sends commands over the bus for the server to execute.
package web

import (
	"net/http"
	"sync"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/mhmdkzr/taskman/internal/app"
	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/routes"
	"github.com/mhmdkzr/taskman/internal/store"
)

// Server is the read-only dashboard. It reads from the shared store and
// streams the shared bus; it never writes anything. phase and diff are the
// transient live state the /events stream learns from bus events and the
// fragment handlers re-render (phase is not persisted in the store).
type Server struct {
	store *store.Store
	pub   publisher.Publisher

	mu     sync.Mutex
	phases map[string]string              // task id -> current pipeline phase
	diffs  map[string]events.PipelineDiff // task id -> latest unified diff
}

// New builds the dashboard from the application's shared dependencies.
func New(a app.App) *Server {
	return &Server{
		store:  a.Deps.Store,
		pub:    a.Deps.Pub,
		phases: make(map[string]string),
		diffs:  make(map[string]events.PipelineDiff),
	}
}

// Routes returns the dashboard's routes, registered on the app mux.
func (s *Server) Routes() []routes.Route {
	return []routes.Route{
		{Method: http.MethodGet, Path: "/", Handler: s.handleIndex},
		{Method: http.MethodGet, Path: "/events", Handler: s.handleEvents},
		{Method: http.MethodGet, Path: "/api/tasks", Handler: s.handleTaskList},
		{Method: http.MethodGet, Path: "/api/tasks/{id}", Handler: s.handleTaskDetail},
	}
}

// setPhase records a task's phase from a bus event.
func (s *Server) setPhase(taskID, phase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phases[taskID] = phase
}

// setDiff records a task's latest diff from a bus event.
func (s *Server) setDiff(taskID string, d events.PipelineDiff) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.diffs[taskID] = d
}

// phase returns the last known phase for a task ("" if none observed).
func (s *Server) phase(taskID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.phases[taskID]
}

// diffFor returns the last known diff for a task (nil if none observed).
func (s *Server) diffFor(taskID string) *events.PipelineDiff {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.diffs[taskID]
	if !ok {
		return nil
	}
	return &d
}

// keep datastar referenced at package level so the import is clearly part of
// the SSE contract even though the handler lives in sse.go.
var _ = datastar.DatastarKey
