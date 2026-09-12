package merged

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// RegisterMCP adds the "task_merged" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store, gitClient *git.Client) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st, gitClient))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_merged",
		Description: "Record the task's merge into its target branch.",
	}
}

func mcpHandler(st *store.Store, gitClient *git.Client) mcp.ToolHandlerFor[Request, task.Task] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Merged(ctx, st, gitClient, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
