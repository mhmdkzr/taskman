package specified

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// RegisterMCP adds the "task_specified" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_specified",
		Description: "Report a task's specification as submitted.",
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, task.Task] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Specified(st, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
