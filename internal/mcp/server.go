// Package mcp exposes loop's tasks over the Model Context Protocol, so an
// external agent harness can read and manage them the same way any other
// caller does. loop assumes nothing about what, if anything, is on the
// other end of this server - it's a plain task tool surface, not a session
// or agent-execution API.
package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/loop/internal/app"
)

// implementation identifies this MCP server to connecting clients.
var implementation = &mcp.Implementation{Name: "loop", Version: "0.1.0"}

// NewServer builds the MCP server exposing loop's task tools.
func NewServer(a app.App) *mcp.Server {
	s := mcp.NewServer(implementation, &mcp.ServerOptions{
		Instructions: "Read loop's tasks: list them, or get one by ID.",
	})
	addTaskTools(s, a)
	return s
}
