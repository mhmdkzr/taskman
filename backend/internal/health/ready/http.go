package ready

import (
	"net/http"

	"github.com/mhmdkzr/app/internal/app"
)

// Handler serves the readiness check endpoint.
type Handler struct {
	app app.App
}

// NewHandler creates a Handler.
func NewHandler(a app.App) Handler {
	return Handler{app: a}
}

// Route returns the route for the readiness check endpoint.
func (h Handler) Route() app.Route {
	return app.Route{
		Method:  http.MethodGet,
		Path:    "/ready",
		Handler: h.handle,
	}
}

func (h Handler) handle(w http.ResponseWriter, r *http.Request) {
	resp := readiness(r.Context(), h.app)
	status := http.StatusOK
	if resp.Status != "ok" {
		status = http.StatusServiceUnavailable
	}
	app.WriteJSON(w, status, resp)
}
