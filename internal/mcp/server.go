// Package mcp assembles taskman's MCP server: each slice's own RegisterMCP
// (under internal/commands/<slice>/mcp.go) wires the same domain function
// its CLI command calls into an MCP tool, over stdio instead of flags/args.
package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/commands/abandon"
	"github.com/mhmdkzr/taskman/internal/commands/commit"
	"github.com/mhmdkzr/taskman/internal/commands/create"
	"github.com/mhmdkzr/taskman/internal/commands/delete"
	"github.com/mhmdkzr/taskman/internal/commands/escalate"
	"github.com/mhmdkzr/taskman/internal/commands/get"
	"github.com/mhmdkzr/taskman/internal/commands/implement"
	"github.com/mhmdkzr/taskman/internal/commands/list"
	"github.com/mhmdkzr/taskman/internal/commands/merge"
	"github.com/mhmdkzr/taskman/internal/commands/next"
	"github.com/mhmdkzr/taskman/internal/commands/prune"
	"github.com/mhmdkzr/taskman/internal/commands/review/approve"
	"github.com/mhmdkzr/taskman/internal/commands/review/record"
	"github.com/mhmdkzr/taskman/internal/commands/review/reject"
	"github.com/mhmdkzr/taskman/internal/commands/specify"
	"github.com/mhmdkzr/taskman/internal/commands/update"
	"github.com/mhmdkzr/taskman/internal/commands/verify"
	"github.com/mhmdkzr/taskman/internal/git"
)

// serverName/serverVersion identify taskman to MCP clients.
const (
	serverName    = "taskman"
	serverVersion = "0.1.0"
)

// NewServer builds the MCP server with every task_* tool registered,
// operating against tasksDir/worktreesDir via git.
func NewServer(tasksDir, worktreesDir string, gitClient *git.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)
	list.RegisterMCP(server, tasksDir)
	get.RegisterMCP(server, tasksDir)
	create.RegisterMCP(server, tasksDir, worktreesDir, gitClient)
	update.RegisterMCP(server, tasksDir)
	specify.RegisterMCP(server, tasksDir)
	implement.RegisterMCP(server, tasksDir)
	verify.RegisterMCP(server, tasksDir)
	record.RegisterMCP(server, tasksDir)
	commit.RegisterMCP(server, tasksDir, gitClient)
	escalate.RegisterMCP(server, tasksDir)
	approve.RegisterMCP(server, tasksDir, gitClient)
	reject.RegisterMCP(server, tasksDir)
	merge.RegisterMCP(server, tasksDir, gitClient)
	abandon.RegisterMCP(server, tasksDir, gitClient)
	next.RegisterMCP(server, tasksDir)
	delete.RegisterMCP(server, tasksDir)
	prune.RegisterMCP(server, tasksDir)
	return server
}
