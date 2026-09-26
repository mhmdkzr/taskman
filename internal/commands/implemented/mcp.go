package implemented

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view/json"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// RegisterMCP adds the "task_implemented" tool to server.
func RegisterMCP(server *mcp.Server, st *store.Store, gitClient *git.Client) {
	mcp.AddTool(server, mcpTool(), mcpHandler(st, gitClient))
}

func mcpTool() *mcp.Tool {
	return &mcp.Tool{
		Name:         "task_implemented",
		Description:  "Report a task's implementation as complete.",
		InputSchema:  utils.SchemaFor[Request](),
		OutputSchema: utils.SchemaFor[json.Document](),
	}
}

func mcpHandler(st *store.Store, gitClient *git.Client) mcp.ToolHandlerFor[Request, json.Document] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, req Request) (*mcp.CallToolResult, json.Document, error) {
		t, err := Implemented(ctx, st, gitClient, req)
		if err != nil {
			return nil, json.Document{}, fmt.Errorf("render task: %w", err)
		}
		doc, err := json.FromTask(t)
		if err != nil {
			return nil, json.Document{}, fmt.Errorf("render task: %w", err)
		}
		return nil, doc, nil
	}
}
