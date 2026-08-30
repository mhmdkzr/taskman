// Package register aggregates metrics route registration.
package register

import (
	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/metrics"
	"github.com/mhmdkzr/app/internal/routes"
)

// RegisterRoutes registers the Prometheus metrics endpoint on the app mux.
func RegisterRoutes(a app.App, m *metrics.Metrics) {
	routes.RegisterRoutes(a,
		metrics.NewHandler(a, m).Route(),
	)
}
