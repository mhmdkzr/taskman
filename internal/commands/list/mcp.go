package list

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view/json"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Result is list's output: the requested page of tasks plus enough to tell
// whether more pages remain.
type Result struct {
	Tasks  []json.Document `json:"tasks"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// RegisterMCP adds the "task_list" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "task_list",
		Description:  "List tasks, optionally filtered by state and labels and paginated.",
		InputSchema:  utils.SchemaFor[Request](),
		OutputSchema: utils.SchemaFor[Result](),
	}
}

func mcpHandler(st *store.Store) mcp.ToolHandlerFor[Request, Result] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, Result, error) {
		tasks, total, err := List(ctx, st, req)
		if err != nil {
			return nil, Result{}, err
		}
		docs := make([]json.Document, 0, len(tasks))
		for _, t := range tasks {
			doc, err := json.FromTask(t)
			if err != nil {
				return nil, Result{}, fmt.Errorf("render task: %w", err)
			}
			docs = append(docs, doc)
		}
		return nil, Result{Tasks: docs, Total: total, Limit: req.Limit, Offset: req.Offset}, nil
	}
}
