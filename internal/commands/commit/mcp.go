package commit

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task"
)

func RegisterMCP(server *mcp.Server, tasksDir string, git *task.GitClient) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir, git))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_commit",
		Description: "Read back a commit already made in the task's worktree.",
	}
}

func mcpHandler(tasksDir string, git *task.GitClient) mcp.ToolHandlerFor[Request, task.Task] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Commit(ctx, tasksDir, git, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
