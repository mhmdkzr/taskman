package opencode

import (
	"net/http"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/routes"
)

// RegisterRoutes registers the OpenCode provider usage endpoint.
func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a,
		routes.Route{
			Method:  http.MethodGet,
			Path:    "/providers/opencode/usage",
			Handler: usageHandler(a),
		},
	)
}
