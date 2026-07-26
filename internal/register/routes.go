package register

import (
	"github.com/mhmdkzr/app/internal/app"
	health "github.com/mhmdkzr/app/internal/health/register"
	"github.com/mhmdkzr/app/pkg/stats"
)

// RegisterRoutes registers all module-level HTTP routes on the app's mux.
func RegisterRoutes(a app.App) {
	health.RegisterRoutes(a)
	stats.RegisterRoutes(a)
}
