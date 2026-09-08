package delete

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterMCP(server *mcp.Server, tasksDir string) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_delete",
		Description: "Remove a task file outright - a human housekeeping action.",
	}
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, any] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, any, error) {
		if err := Delete(tasksDir, req); err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("deleted task %s", req.ID)}},
		}, nil, nil
	}
}
