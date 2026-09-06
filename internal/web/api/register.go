// Package api registers the web UI's HTTP routes: full-page session views
// for direct navigation, and the Datastar actions (create session, send a
// message) that patch them in place.
package api

import (
	"net/http"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/routes"
)

// RegisterRoutes registers the session pages and the actions that mutate them.
func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a,
		routes.Route{Method: http.MethodGet, Path: "/", Handler: indexHandler(a)},
		routes.Route{Method: http.MethodGet, Path: "/new", Handler: newSessionPageHandler(a)},
		routes.Route{Method: http.MethodGet, Path: "/sessions/refresh", Handler: refreshIndexHandler(a)},
		routes.Route{Method: http.MethodGet, Path: "/sessions/{id}", Handler: sessionPageHandler(a)},
		routes.Route{Method: http.MethodGet, Path: "/sessions/{id}/refresh", Handler: refreshSessionHandler(a)},
		routes.Route{Method: http.MethodPost, Path: "/sessions", Handler: createSessionHandler(a)},
		routes.Route{Method: http.MethodPost, Path: "/sessions/{id}/messages", Handler: respondHandler(a)},
		routes.Route{
			Method: http.MethodPost, Path: "/sessions/{id}/asks/{ask_id}/answer", Handler: answerAskHandler(a),
		},
		routes.Route{Method: http.MethodGet, Path: "/tasks", Handler: tasksPageHandler(a)},
		routes.Route{Method: http.MethodGet, Path: "/tasks/refresh", Handler: refreshTasksHandler(a)},
		routes.Route{Method: http.MethodGet, Path: "/tasks/{id}", Handler: taskPageHandler(a)},
		routes.Route{Method: http.MethodGet, Path: "/tasks/{id}/refresh", Handler: refreshTaskHandler(a)},
	)
}
