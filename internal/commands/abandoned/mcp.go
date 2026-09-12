package abandoned

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view/json"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// RegisterMCP adds the "task_abandoned" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "task_abandoned",
		Description:  "End a task unsuccessfully.",
		InputSchema:  utils.SchemaFor[Request](),
		OutputSchema: utils.SchemaFor[json.Document](),
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, json.Document] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, json.Document, error) {
		t, err := Abandoned(ctx, st, req)
		if err != nil {
			return nil, json.Document{}, err
		}
		return nil, json.FromTask(t), nil
	}
}
