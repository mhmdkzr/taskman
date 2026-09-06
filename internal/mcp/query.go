package mcp

import (
	"context"
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/loop/internal/agent/tools/query"
	"github.com/mhmdkzr/loop/internal/app"
)

// addQueryTools registers read-only SQL access, reusing
// internal/agent/tools/query's own Execute directly (which itself rejects
// anything but SELECT/WITH/EXPLAIN).
func addQueryTools(s *gomcp.Server, a app.App) {
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        query.Name,
		Description: query.Description,
	}, queryHandler(a))
}

// queryInput mirrors query.Input with MCP-flavored jsonschema tags - see the
// comment on historyInput for why this isn't query.Input directly.
type queryInput struct {
	Query      string `json:"query"                  jsonschema:"Read-only SQL SELECT statement."`
	MaxCellLen *int   `json:"max_cell_len,omitempty" jsonschema:"Maximum characters rendered per cell (default 200, max 10000)."`
}

func queryHandler(a app.App) gomcp.ToolHandlerFor[queryInput, query.Output] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in queryInput) (*gomcp.CallToolResult, query.Output, error) {
		qin := query.Input{Query: in.Query, MaxCellLen: in.MaxCellLen}
		if err := qin.Validate(); err != nil {
			return nil, query.Output{}, fmt.Errorf("query: %w", err)
		}
		out, err := query.Execute(ctx, a.Deps.Store, qin)
		if err != nil {
			return nil, query.Output{}, fmt.Errorf("query: %w", err)
		}
		return nil, out, nil
	}
}
