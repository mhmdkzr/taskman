// Package mcp assembles taskman's MCP server: every command slice's tool,
// registered onto one *mcp.Server.
package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/commands/abandoned"
	"github.com/mhmdkzr/taskman/internal/commands/committed"
	"github.com/mhmdkzr/taskman/internal/commands/create"
	"github.com/mhmdkzr/taskman/internal/commands/escalated"
	"github.com/mhmdkzr/taskman/internal/commands/get"
	implementationreviewagentapproved "github.com/mhmdkzr/taskman/internal/commands/implementation/review/agent/approved"
	implementationreviewagentrejected "github.com/mhmdkzr/taskman/internal/commands/implementation/review/agent/rejected"
	implementationreviewhumanapproved "github.com/mhmdkzr/taskman/internal/commands/implementation/review/human/approved"
	implementationreviewhumanrejected "github.com/mhmdkzr/taskman/internal/commands/implementation/review/human/rejected"
	"github.com/mhmdkzr/taskman/internal/commands/implemented"
	"github.com/mhmdkzr/taskman/internal/commands/list"
	"github.com/mhmdkzr/taskman/internal/commands/merged"
	"github.com/mhmdkzr/taskman/internal/commands/next"
	specificationreviewagentapproved "github.com/mhmdkzr/taskman/internal/commands/specification/review/agent/approved"
	specificationreviewagentrejected "github.com/mhmdkzr/taskman/internal/commands/specification/review/agent/rejected"
	specificationreviewhumanapproved "github.com/mhmdkzr/taskman/internal/commands/specification/review/human/approved"
	specificationreviewhumanrejected "github.com/mhmdkzr/taskman/internal/commands/specification/review/human/rejected"
	"github.com/mhmdkzr/taskman/internal/commands/specified"
	"github.com/mhmdkzr/taskman/internal/commands/verified"
	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

const (
	serverName    = "taskman"
	serverVersion = "0.1.0"
)

// NewServer builds an MCP server exposing every command as a tool. st and
// gitClient are shared across every tool call for the server's lifetime.
func NewServer(st *store.Store, gitClient *git.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: serverName, Version: serverVersion}, nil)

	create.RegisterMCP(server, st)
	get.RegisterMCP(server, st)
	list.RegisterMCP(server, st)
	next.RegisterMCP(server, st)
	specified.RegisterMCP(server, st)
	specificationreviewagentapproved.RegisterMCP(server, st)
	specificationreviewagentrejected.RegisterMCP(server, st)
	specificationreviewhumanapproved.RegisterMCP(server, st)
	specificationreviewhumanrejected.RegisterMCP(server, st)
	implemented.RegisterMCP(server, st)
	verified.RegisterMCP(server, st)
	implementationreviewagentapproved.RegisterMCP(server, st)
	implementationreviewagentrejected.RegisterMCP(server, st)
	implementationreviewhumanapproved.RegisterMCP(server, st)
	implementationreviewhumanrejected.RegisterMCP(server, st)
	committed.RegisterMCP(server, st, gitClient)
	merged.RegisterMCP(server, st, gitClient)
	escalated.RegisterMCP(server, st)
	abandoned.RegisterMCP(server, st)

	return server
}
