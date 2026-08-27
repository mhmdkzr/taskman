// Package register aggregates health route registration.
package register

import (
	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/health/get"
	"github.com/mhmdkzr/app/internal/health/ready"
	"github.com/mhmdkzr/app/internal/routes"
)

func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a,
		get.NewHandler(a).Route(),
		ready.NewHandler(a).Route(),
	)
}
