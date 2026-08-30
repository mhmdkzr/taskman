// Package register aggregates authentication routes.
package register

import (
	"net/http"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/auth/bff"
	"github.com/mhmdkzr/app/internal/auth/loginui"
	"github.com/mhmdkzr/app/internal/routes"
)

// RegisterRoutes registers BFF authentication routes, and the custom
// Session-API login UI routes when configured, only when auth is configured.
func RegisterRoutes(a app.App, service *bff.Service, loginUI *loginui.Service) {
	if service == nil {
		return
	}
	h := bff.NewHandler(service)
	routes.RegisterRoutes(
		a,
		routes.Route{
			Method:  http.MethodGet,
			Path:    "/auth/login",
			Handler: service.TransactionMiddleware(http.HandlerFunc(h.Login)).ServeHTTP,
		},
		routes.Route{
			Method:  http.MethodGet,
			Path:    "/auth/callback",
			Handler: service.TransactionMiddleware(http.HandlerFunc(h.Callback)).ServeHTTP,
		},
		routes.Route{Method: http.MethodPost, Path: "/auth/logout", Handler: h.Logout},
		routes.Route{Method: http.MethodGet, Path: "/api/me", Handler: h.Me},
	)
	if loginUI == nil {
		return
	}
	lh := loginui.NewHandler(loginUI)
	wrap := func(handler http.HandlerFunc) http.HandlerFunc {
		return loginUI.TransactionMiddleware(handler).ServeHTTP
	}
	// Finalize calls into bff.Service.CompleteAuthorizationCallback, which
	// reads bff's own auth_transaction-backed context (state, PKCE verifier)
	// set up by GET /auth/login — so it needs bff's TransactionMiddleware
	// loaded too, not just loginUI's login_session one.
	wrapWithBFFTransaction := func(handler http.HandlerFunc) http.HandlerFunc {
		return service.TransactionMiddleware(loginUI.TransactionMiddleware(handler)).ServeHTTP
	}
	routes.RegisterRoutes(
		a,
		routes.Route{Method: http.MethodPost, Path: "/auth/session", Handler: wrap(lh.CreateSession)},
		routes.Route{Method: http.MethodPatch, Path: "/auth/session/password", Handler: wrap(lh.CheckPassword)},
		routes.Route{Method: http.MethodPatch, Path: "/auth/session/totp", Handler: wrap(lh.CheckTOTP)},
		routes.Route{
			Method: http.MethodPost, Path: "/auth/session/finalize", Handler: wrapWithBFFTransaction(lh.Finalize),
		},
	)

	rh := loginui.NewRegistrationHandler(loginUI)
	routes.RegisterRoutes(
		a,
		routes.Route{Method: http.MethodPost, Path: "/auth/register", Handler: rh.Register},
		routes.Route{Method: http.MethodPost, Path: "/auth/verify-email", Handler: rh.VerifyEmail},
		routes.Route{Method: http.MethodPost, Path: "/auth/password-reset", Handler: rh.RequestPasswordReset},
		routes.Route{Method: http.MethodPost, Path: "/auth/password-reset/confirm", Handler: rh.ResetPassword},
		routes.Route{Method: http.MethodPost, Path: "/auth/mfa/totp", Handler: rh.StartTOTPEnrollment},
		routes.Route{Method: http.MethodPost, Path: "/auth/mfa/totp/verify", Handler: rh.ConfirmTOTPEnrollment},
	)
}
