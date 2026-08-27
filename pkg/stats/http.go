package stats

import (
	"net/http"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/routes"
	"github.com/mhmdkzr/app/pkg/jsonresp"
)

// readStats is a seam over Read so tests can exercise the handler's error path
// without depending on the host's actual resource-stats availability.
var readStats = Read

func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a, routes.Route{
		Method:  http.MethodGet,
		Path:    "/stats",
		Handler: Handler(),
	})
}

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := readStats()
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		jsonresp.WriteJSON(w, http.StatusOK, s)
	}
}
