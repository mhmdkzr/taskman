// Package register aggregates frontend route registration.
package register

import (
	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/frontend/serve"
)

func RegisterRoutes(a app.App) {
	a.RegisterRoutes(serve.NewHandler().Route())
}
