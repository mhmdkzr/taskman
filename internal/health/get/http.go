package get

import (
	"net/http"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/routes"
	"github.com/mhmdkzr/app/pkg/jsonresp"
)

// Handler serves the health check endpoint.
type Handler struct{}

// NewHandler creates a Handler.
func NewHandler(_ app.App) Handler {
	return Handler{}
}

func (h Handler) Route() routes.Route {
	return routes.Route{
		Method:  http.MethodGet,
		Path:    "/health",
		Handler: h.handle,
	}
}

// handle writes the health status response.
func (h Handler) handle(w http.ResponseWriter, r *http.Request) {
	healthResp := get()
	jsonresp.WriteJSON(w, http.StatusOK, healthResp)
}
