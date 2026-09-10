package merge

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
)

func RegisterMCP(server *mcp.Server, tasksDir string, gitClient *git.Client) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir, gitClient))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_merged",
		Description: "Record that a task's branch was already merged.",
	}
}

func mcpHandler(tasksDir string, gitClient *git.Client) mcp.ToolHandlerFor[Request, task.Task] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Merge(ctx, tasksDir, gitClient, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
