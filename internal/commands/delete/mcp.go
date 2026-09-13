package delete

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// RegisterMCP adds the "task_delete" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "task_delete",
		Description:  "Permanently delete a task and its event log. Returns the id of the deleted task.",
		InputSchema:  utils.SchemaFor[Request](),
		OutputSchema: utils.SchemaFor[Result](),
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, Result] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, Result, error) {
		result, err := Delete(ctx, st, req)
		if err != nil {
			return nil, Result{}, err
		}
		return nil, result, nil
	}
}
