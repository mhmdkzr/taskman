// Package app holds the shared runtime dependencies (App.Deps), configuration
// (App.Cfg), and HTTP mux (App.Mux) that slices register against.
package app

import (
	"database/sql"
	"net/http"

	agenttools "github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/app/config"
)

// App bundles the application dependencies, configuration, and HTTP mux.
type App struct {
	Deps Deps
	Cfg  config.Config
	Mux  *http.ServeMux
}

// Deps holds the shared runtime dependencies of the application.
type Deps struct {
	DB         *sql.DB
	AgentTools agenttools.Deps
}
