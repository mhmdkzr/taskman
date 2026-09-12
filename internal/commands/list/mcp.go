package list

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view/json"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Request is list's (empty) input.
type Request struct{}

// RegisterMCP adds the "task_list" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "task_list",
		Description:  "List every task.",
		InputSchema:  utils.SchemaFor[Request](),
		OutputSchema: utils.SchemaFor[[]json.Document](),
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, []json.Document] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ Request) (*mcp.CallToolResult, []json.Document, error) {
		tasks, err := List(ctx, st)
		if err != nil {
			return nil, nil, err
		}
		docs := make([]json.Document, 0, len(tasks))
		for _, t := range tasks {
			docs = append(docs, json.FromTask(t))
		}
		return nil, docs, nil
	}
}
