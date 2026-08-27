// Package register aggregates frontend route registration.
package register

import (
	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/frontend/serve"
	"github.com/mhmdkzr/app/internal/routes"
)

func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a, serve.NewHandler().Route())
}
