package migrate

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterMCP(server *mcp.Server, tasksDir string) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "task_migrate", Description: "Migrate legacy task files to the current schema.",
	}, mcpHandler(tasksDir))
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, Result] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, Result, error) {
		result, err := Migrate(tasksDir, req)
		return nil, result, err
	}
}
