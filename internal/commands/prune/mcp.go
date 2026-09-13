package prune

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// RegisterMCP adds the "task_prune" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "task_prune",
		Description:  "Permanently delete every completed task and its event log. With dry-run, report what would be deleted without deleting. Returns the pruned task ids.",
		InputSchema:  utils.SchemaFor[Request](),
		OutputSchema: utils.SchemaFor[Result](),
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, Result] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, Result, error) {
		result, err := Prune(ctx, st, req)
		if err != nil {
			return nil, Result{}, err
		}
		return nil, result, nil
	}
}
