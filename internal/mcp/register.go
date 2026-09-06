package mcp

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/routes"
)

// RegisterRoutes mounts the MCP endpoint at /mcp, on the same HTTP server and
// port as the web UI - so operating loop over MCP and watching it in the
// browser are the same instance, the same data, no second port to run.
//
// The streamable HTTP transport dispatches GET/POST/DELETE itself, so all
// three are registered as explicit method+path routes.Route patterns (rather
// than routes.HandleAny's bare "/mcp", which Go's ServeMux rejects at
// registration time as ambiguous against web's "GET /" subtree pattern).
func RegisterRoutes(a app.App) {
	server := NewServer(a)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	routes.RegisterRoutes(a,
		routes.Route{Method: http.MethodGet, Path: "/mcp", Handler: handler.ServeHTTP},
		routes.Route{Method: http.MethodPost, Path: "/mcp", Handler: handler.ServeHTTP},
		routes.Route{Method: http.MethodDelete, Path: "/mcp", Handler: handler.ServeHTTP},
		// MCP clients probe these well-known URLs to decide whether the
		// server requires OAuth. Without an explicit route here, they fall
		// through to the web UI's "GET /" subtree pattern and get back its
		// HTML page instead of a 404 - which some clients then fail to
		// parse as JSON and mistake for a broken auth flow. loop has no
		// auth, so answer plainly that these resources don't exist.
		routes.Route{Method: http.MethodGet, Path: "/.well-known/oauth-protected-resource", Handler: notFound},
		routes.Route{Method: http.MethodGet, Path: "/.well-known/oauth-authorization-server", Handler: notFound},
	)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
