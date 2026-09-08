package approve

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task"
)

func RegisterMCP(server *mcp.Server, tasksDir string) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_review_approve",
		Description: "Record a human's approval at the review stage. Only call this on a task's own behalf when explicitly told to (e.g. --auto-approve) - never approve a human review yourself otherwise.",
	}
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, task.Task] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := ApproveReview(tasksDir, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
