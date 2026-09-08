package register

import (
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/mcp"
	"github.com/mhmdkzr/loop/internal/providers/opencode"
)

// RegisterRoutes registers all module-level HTTP routes on the app's mux.
func RegisterRoutes(a app.App) {
	mcp.RegisterRoutes(a)
	opencode.RegisterRoutes(a)
}
