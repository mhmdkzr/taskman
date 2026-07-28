// Package serve provides the frontend static file handler.
package serve

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/mhmdkzr/app/frontend"
	"github.com/mhmdkzr/app/internal/app"
)

type Handler struct {
	fileServer http.Handler
	subFS      fs.FS
}

func NewHandler() *Handler {
	sub, err := fs.Sub(frontend.FS, "dist")
	if err != nil {
		panic("embedded frontend not found: " + err.Error())
	}
	return &Handler{
		fileServer: http.FileServer(http.FS(sub)),
		subFS:      sub,
	}
}

func (h *Handler) Route() app.Route {
	return app.Route{
		Method:  http.MethodGet,
		Path:    "/",
		Handler: h.ServeHTTP,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path != "" {
		if _, err := fs.Stat(h.subFS, path); err != nil {
			r.URL.Path = "/"
		}
	}
	h.fileServer.ServeHTTP(w, r)
}
