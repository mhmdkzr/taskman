package register

import (
	"github.com/mhmdkzr/taskman/internal/app"
	health "github.com/mhmdkzr/taskman/internal/health/register"
	"github.com/mhmdkzr/taskman/internal/routes"
	"github.com/mhmdkzr/taskman/internal/web"
)

// RegisterRoutes registers all module-level HTTP routes on the app's mux.
func RegisterRoutes(a app.App) {
	health.RegisterRoutes(a)
	routes.RegisterRoutes(a, web.New(a).Routes()...)
}
