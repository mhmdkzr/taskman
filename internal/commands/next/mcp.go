package next

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// RegisterMCP adds the "task_next" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "task_next",
		Description:  "Show what should happen next for a task, and the command(s) that report it.",
		InputSchema:  utils.SchemaFor[Request](),
		OutputSchema: utils.SchemaFor[Guidance](),
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, Guidance] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, Guidance, error) {
		guidance, err := Next(ctx, st, req)
		if err != nil {
			return nil, Guidance{}, err
		}
		return nil, guidance, nil
	}
}
