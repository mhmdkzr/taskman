package register

import (
	"github.com/mhmdkzr/app/internal/app"
	frontend "github.com/mhmdkzr/app/internal/frontend/register"
	health "github.com/mhmdkzr/app/internal/health/register"
	"github.com/mhmdkzr/app/internal/metrics"
	metricsregister "github.com/mhmdkzr/app/internal/metrics/register"
)

// RegisterRoutes registers all module-level HTTP routes on the app's mux.
func RegisterRoutes(a app.App, m *metrics.Metrics) {
	health.RegisterRoutes(a)
	frontend.RegisterRoutes(a)
	metricsregister.RegisterRoutes(a, m)
}
