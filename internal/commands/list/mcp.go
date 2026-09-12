package list

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is list's (empty) input.
type Request struct{}

// RegisterMCP adds the "task_list" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_list",
		Description: "List every task.",
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, []task.Task] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ Request) (*mcp.CallToolResult, []task.Task, error) {
		tasks, err := List(st)
		if err != nil {
			return nil, nil, err
		}
		return nil, tasks, nil
	}
}
