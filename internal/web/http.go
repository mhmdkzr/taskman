// Package web registers the application's web interface.
package web

import (
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/web/api"
)

// RegisterRoutes registers the web interface routes.
func RegisterRoutes(a app.App) {
	api.RegisterRoutes(a)
}
