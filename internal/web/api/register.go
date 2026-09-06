// Package api registers the web UI's one HTTP route: the task list at "/" -
// the whole UI (every task's full detail, each session dispatched for it,
// and every turn of that session's activity) on one page, including the
// Datastar action that keeps it live (see indexHandler).
package api

import (
	"net/http"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/routes"
)

// RegisterRoutes registers the task list.
func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a,
		routes.Route{Method: http.MethodGet, Path: "/", Handler: indexHandler(a)},
	)
}
