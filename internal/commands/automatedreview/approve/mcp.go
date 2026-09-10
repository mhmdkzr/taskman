package approve

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task"
)

// RegisterMCP registers the automated-review approval tool.
func RegisterMCP(server *mcp.Server, tasksDir string) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{Name: "task_automated_review_approved", Description: "Record an automated review approval."}
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, task.Task] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Approve(tasksDir, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
