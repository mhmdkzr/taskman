package metrics

import (
	"net/http"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/routes"
)

// HandlerHTTP serves the Prometheus metrics endpoint.
type HandlerHTTP struct {
	metrics *Metrics
}

// NewHandler creates a HandlerHTTP.
func NewHandler(_ app.App, m *Metrics) HandlerHTTP {
	return HandlerHTTP{metrics: m}
}

// Route returns the metrics route definition.
func (h HandlerHTTP) Route() routes.Route {
	return routes.Route{
		Method:  http.MethodGet,
		Path:    "/metrics",
		Handler: h.handle,
	}
}

func (h HandlerHTTP) handle(w http.ResponseWriter, r *http.Request) {
	h.metrics.Handler().ServeHTTP(w, r)
}
