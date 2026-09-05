package register

import (
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/routes"
	"github.com/mhmdkzr/loop/internal/web"
)

// RegisterRoutes registers all module-level HTTP routes on the app's mux.
func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a)
	web.RegisterRoutes(a)
}
