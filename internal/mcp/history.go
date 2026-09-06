package mcp

import (
	"context"
	"fmt"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/loop/internal/agent/tools/history"
	"github.com/mhmdkzr/loop/internal/app"
)

// addHistoryTools registers read-only search over past conversation turns,
// reusing internal/agent/tools/history's own Execute directly.
func addHistoryTools(s *gomcp.Server, a app.App) {
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        history.Name,
		Description: history.Description,
	}, historyHandler(a))
}

// historyInput mirrors history.Input, field for field: the MCP SDK's
// jsonschema-go inference expects a bare description in the jsonschema tag
// (see WeatherInput in its own examples), not this repo's
// `jsonschema:"description=..."` convention that goai.SchemaFrom parses
// elsewhere - so this type carries the MCP-flavored tags and is mapped onto
// history.Input before calling Execute, rather than reusing history.Input's
// struct tags directly.
type historyInput struct {
	SessionID  string `json:"session_id,omitempty"   jsonschema:"Session ID filter."`
	Query      string `json:"query,omitempty"        jsonschema:"Substring to find in prompts or replies."`
	Limit      *int   `json:"limit,omitempty"        jsonschema:"Maximum turns to return (default 10, max 50)."`
	MaxCellLen *int   `json:"max_cell_len,omitempty" jsonschema:"Maximum rendered characters (default 200, max 10000)."`
}

func historyHandler(a app.App) gomcp.ToolHandlerFor[historyInput, history.Output] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in historyInput) (*gomcp.CallToolResult, history.Output, error) {
		hin := history.Input{
			SessionID: in.SessionID, Query: in.Query, Limit: in.Limit, MaxCellLen: in.MaxCellLen,
		}
		if err := hin.Validate(); err != nil {
			return nil, history.Output{}, fmt.Errorf("history: %w", err)
		}
		out, err := history.Execute(ctx, a.Deps.Store, hin)
		if err != nil {
			return nil, history.Output{}, fmt.Errorf("history: %w", err)
		}
		return nil, out, nil
	}
}
