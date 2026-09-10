package specificationreject

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task"
)

// RegisterMCP registers the specification-rejection tool.
func RegisterMCP(server *mcp.Server, tasksDir string) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_specification_rejected",
		Description: "Record a human's rejection of a drafted specification. Never call this on a human's behalf.",
	}
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, task.Task] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Reject(tasksDir, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
