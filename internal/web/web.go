// Package web serves a read-only, realtime dashboard for the task backlog and
// the agent bus: it renders tasks and sessions from the SQLite store and
// streams live agent/pipeline events over Server-Sent Events. It is purely
// observational — there is no control surface here. Controls live in the
// taskman CLI, which sends commands over the bus for the server to execute.
package web

import (
	"net/http"

	"github.com/mhmdkzr/taskman/internal/app"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/routes"
	"github.com/mhmdkzr/taskman/internal/store"
)

// Server is the read-only dashboard. It reads from the shared store and
// streams the shared bus; it never writes anything.
type Server struct {
	store *store.Store
	pub   publisher.Publisher
}

// New builds the dashboard from the application's shared dependencies.
func New(a app.App) *Server {
	return &Server{store: a.Deps.Store, pub: a.Deps.Pub}
}

// Routes returns the dashboard's routes, registered on the app mux.
func (s *Server) Routes() []routes.Route {
	return []routes.Route{
		{Method: http.MethodGet, Path: "/", Handler: s.handleIndex},
		{Method: http.MethodGet, Path: "/events", Handler: s.handleEvents},
		{Method: http.MethodGet, Path: "/api/overview", Handler: s.handleOverview},
	}
}
