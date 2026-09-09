package prune

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterMCP(server *mcp.Server, tasksDir string) {
	mcp.AddTool(server, mcpTool(), mcpHandler(tasksDir))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "task_prune",
		Description: "Delete every completed task file - a human housekeeping action.",
	}
}

func mcpHandler(tasksDir string) mcp.ToolHandlerFor[Request, any] {
	return func(_ context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, any, error) {
		res, err := Prune(tasksDir, req)
		if err != nil {
			return nil, nil, err
		}
		data, err := json.Marshal(res)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal result: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	}
}
