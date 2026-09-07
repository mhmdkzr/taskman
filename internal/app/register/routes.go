package register

import (
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/routes"
	"github.com/mhmdkzr/loop/internal/mcp"
	"github.com/mhmdkzr/loop/internal/providers/opencode"
	"github.com/mhmdkzr/loop/internal/web/api"
)

// RegisterRoutes registers all module-level HTTP routes on the app's mux.
func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a)
	api.RegisterRoutes(a)
	mcp.RegisterRoutes(a)
	opencode.RegisterRoutes(a)
}
