// Package register aggregates health route registration.
package register

import (
	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/health/get"
	"github.com/mhmdkzr/app/internal/health/ready"
)

func RegisterRoutes(a app.App) {
	a.RegisterRoutes(
		get.NewHandler(a).Route(),
		ready.NewHandler(a).Route(),
	)
}
