// Package register aggregates authentication routes.
package register

import (
	"net/http"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/auth/bff"
	"github.com/mhmdkzr/app/internal/routes"
)

// RegisterRoutes registers BFF authentication routes only when auth is configured.
func RegisterRoutes(a app.App, service *bff.Service) {
	if service == nil {
		return
	}
	h := bff.NewHandler(service)
	routes.RegisterRoutes(a,
		routes.Route{Method: http.MethodGet, Path: "/auth/login", Handler: service.TransactionMiddleware(http.HandlerFunc(h.Login)).ServeHTTP},
		routes.Route{Method: http.MethodGet, Path: "/auth/callback", Handler: service.TransactionMiddleware(http.HandlerFunc(h.Callback)).ServeHTTP},
		routes.Route{Method: http.MethodPost, Path: "/auth/logout", Handler: h.Logout},
		routes.Route{Method: http.MethodGet, Path: "/api/me", Handler: h.Me},
	)
}
