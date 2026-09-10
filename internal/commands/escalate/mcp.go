package escalate

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
		Name:        "task_escalated",
		Description: "Block a task because a dispatched agent gave up.",
	}
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, task.Task] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, task.Task, error) {
		t, err := Escalate(tasksDir, req)
		if err != nil {
			return nil, task.Task{}, err
		}
		return nil, t, nil
	}
}
