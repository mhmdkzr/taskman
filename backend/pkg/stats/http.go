package stats

import (
	"net/http"

	"github.com/mhmdkzr/app/internal/app"
)

func RegisterRoutes(a app.App) {
	a.RegisterRoutes(app.Route{
		Method:  http.MethodGet,
		Path:    "/stats",
		Handler: Handler(),
	})
}

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := Read()
		if err != nil {
			app.WriteHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		app.WriteJSON(w, http.StatusOK, s)
	}
}
