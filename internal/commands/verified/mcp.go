package verified

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// RegisterMCP adds the "task_verified" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_verified",
		Description: "Report one verification attempt.",
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, task.Task] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Verified(st, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
