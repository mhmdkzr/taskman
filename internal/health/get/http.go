package get

import (
	"net/http"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/pkg/resp"
)

// Handler serves the health check endpoint.
type Handler struct{}

// NewHandler creates a Handler.
func NewHandler(_ app.App) Handler {
	return Handler{}
}

func (h Handler) Route() app.Route {
	return app.Route{
		Method:  http.MethodGet,
		Path:    "/health",
		Handler: h.handle,
	}
}

// handle writes the health status response.
func (h Handler) handle(w http.ResponseWriter, r *http.Request) {
	healthResp := get()
	resp.WriteJSON(w, http.StatusOK, healthResp)
}
