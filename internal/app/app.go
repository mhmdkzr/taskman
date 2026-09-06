// Package app holds the shared runtime dependencies (App.Deps), configuration
// (App.Cfg), and HTTP mux (App.Mux) that slices register against.
package app

import (
	"net/http"

	agenttools "github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
)

// App bundles the application dependencies, configuration, and HTTP mux.
type App struct {
	Deps Deps
	Cfg  config.Config
	Mux  *http.ServeMux
}

// Deps holds the shared runtime dependencies of the application.
type Deps struct {
	Store      *store.Store
	AgentTools agenttools.Deps
}
