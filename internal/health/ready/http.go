package ready

import (
	"net/http"

	"github.com/mhmdkzr/taskman/internal/app"
	"github.com/mhmdkzr/taskman/internal/routes"
	"github.com/mhmdkzr/taskman/pkg/jsonresp"
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
func (h Handler) Route() routes.Route {
	return routes.Route{
		Method:  http.MethodGet,
		Path:    "/ready",
		Handler: h.handle,
	}
}

func (h Handler) handle(w http.ResponseWriter, r *http.Request) {
	readyResp := readiness(r.Context(), h.app)
	status := http.StatusOK
	if readyResp.Status != "ok" {
		status = http.StatusServiceUnavailable
	}
	jsonresp.WriteJSON(w, status, readyResp)
}
