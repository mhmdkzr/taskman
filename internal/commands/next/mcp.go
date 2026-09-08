package next

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterMCP(server *mcp.Server, tasksDir string) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_next",
		Description: "Show what should happen next for a task - the core driver loop.",
	}
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, Guidance] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, Guidance, error) {
		g, err := Next(tasksDir, req)
		if err != nil {
			return nil, Guidance{}, err
		}
		return nil, g, nil
	}
}
