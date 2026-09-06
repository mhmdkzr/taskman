// Package mcp exposes loop's own agent sessions over the Model Context
// Protocol, so an external agent (including one operating loop itself) can
// drive them the same way the web UI does: create a session, prompt it,
// answer a question it asks back, and read its progress. Every other tool
// package under internal/agent/tools stays internal to loop's own agents -
// this package deliberately does not expose bash, file, or other execution
// tools to an external MCP client. Everything except the session tools is
// read-only.
package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/loop/internal/app"
)

// implementation identifies this MCP server to connecting clients.
var implementation = &mcp.Implementation{Name: "loop", Version: "0.1.0"}

// NewServer builds the MCP server exposing loop's read/write session tools
// and its read-only agent, history, query, tool, and task tools.
func NewServer(a app.App) *mcp.Server {
	s := mcp.NewServer(implementation, &mcp.ServerOptions{
		Instructions: "Operate loop's own agent sessions: create a session, prompt it, " +
			"answer any question it asks back (session_get reports a pending_ask), and " +
			"read its progress. All other tools here (agent, history, query, tool, task) " +
			"are read-only lookups to help you decide what to do.",
	})
	addSessionTools(s, a)
	addAgentTools(s, a)
	addHistoryTools(s, a)
	addQueryTools(s, a)
	addToolTools(s, a)
	addTaskTools(s, a)
	return s
}
