package list

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterMCP(server *mcp.Server, tasksDir string) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_list",
		Description: "List tasks, optionally filtered by state/label and paginated.",
	}
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, Result] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, Result, error) {
		if req.Limit == 0 {
			req.Limit = defaultLimit
		}
		result, err := List(tasksDir, req)
		if err != nil {
			return nil, Result{}, err
		}
		return nil, result, nil
	}
}
