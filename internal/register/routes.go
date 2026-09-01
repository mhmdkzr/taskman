package register

import (
	"github.com/mhmdkzr/taskman/internal/app"
	health "github.com/mhmdkzr/taskman/internal/health/register"
)

// RegisterRoutes registers all module-level HTTP routes on the app's mux.
func RegisterRoutes(a app.App) {
	health.RegisterRoutes(a)
}
