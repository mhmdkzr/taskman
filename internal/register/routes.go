package register

import (
	"github.com/mhmdkzr/app/internal/app"
	frontend "github.com/mhmdkzr/app/internal/frontend/register"
	health "github.com/mhmdkzr/app/internal/health/register"
)

// RegisterRoutes registers all module-level HTTP routes on the app's mux.
func RegisterRoutes(a app.App) {
	health.RegisterRoutes(a)
	frontend.RegisterRoutes(a)
}
