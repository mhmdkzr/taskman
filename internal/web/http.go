// Package web registers the application's web interface.
package web

import (
	"bytes"
	"log/slog"
	"net/http"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/routes"
	"github.com/mhmdkzr/loop/internal/web/components"
)

// RegisterRoutes registers the web interface routes.
func RegisterRoutes(a app.App) {
	routes.RegisterRoutes(a,
		routes.Route{Method: http.MethodGet, Path: "/", Handler: homeHandler},
		routes.Route{Method: http.MethodPost, Path: "/", Handler: homeUpdateHandler},
	)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if err := components.Home().Render(r.Context(), w); err != nil {
		slog.Error("render home page", "error", err)
	}
}

func homeUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var html bytes.Buffer
	if err := components.Home().Render(r.Context(), &html); err != nil {
		http.Error(w, "failed to render home page", http.StatusInternalServerError)
		slog.Error("render home update", "error", err)
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.PatchElements(html.String()); err != nil {
		slog.Error("patch home page", "error", err)
	}
}
