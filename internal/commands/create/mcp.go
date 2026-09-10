package create

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
)

func RegisterMCP(server *mcp.Server, tasksDir, worktreesDir string, gitClient *git.Client) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir, worktreesDir, gitClient))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_create",
		Description: "Create a task and its worktree/branch (or set trunk to work it on the current branch).",
	}
}

func mcpHandler(tasksDir, worktreesDir string, gitClient *git.Client) mcp.ToolHandlerFor[Request, task.Task] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Create(ctx, tasksDir, worktreesDir, gitClient, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
