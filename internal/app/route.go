package app

import (
	"net/http"
	"path"
	"strings"
)

// Route describes a single HTTP route with its method, path, and handler.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// RegisterRoutes registers each route on the application HTTP mux, prepending
// the configured BasePath to each path.
func (a App) RegisterRoutes(routes ...Route) {
	for _, route := range routes {
		a.Handle(route.Method, route.Path, route.Handler)
	}
}

func (a App) Handle(method, routePath string, handler http.HandlerFunc) {
	basePath := strings.TrimSpace(a.Cfg.Server.BasePath)
	if basePath == "" {
		basePath = "/"
	}
	pattern := method + " " + joinBasePath(basePath, routePath)
	a.Mux.HandleFunc(pattern, handler)
}

func joinBasePath(basePath, routePath string) string {
	if basePath == "/" {
		if strings.HasPrefix(routePath, "/") {
			return routePath
		}
		return "/" + routePath
	}

	if routePath == "" {
		return path.Clean(basePath)
	}

	return path.Join(basePath, routePath)
}
