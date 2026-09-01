// Package register aggregates health route registration.
package register

import (
	"github.com/mhmdkzr/taskman/internal/app"
	"github.com/mhmdkzr/taskman/internal/health/get"
	"github.com/mhmdkzr/taskman/internal/health/ready"
	"github.com/mhmdkzr/taskman/internal/routes"
)

func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a,
		get.NewHandler(a).Route(),
		ready.NewHandler(a).Route(),
	)
}
