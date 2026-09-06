package history

import (
	"context"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "history"
	Description = "Read past conversation turns across agent sessions, newest first. " +
		"Search first: pass query to find turns whose prompt or reply contains it. " +
		"To focus on one session pass its session_id (ids appear in the output). " +
		"Bound limit to cap turns returned."
)

type Input struct {
	SessionID  string `json:"session_id,omitempty"   jsonschema:"description=Session ID filter."`
	Query      string `json:"query,omitempty"        jsonschema:"description=Substring to find in prompts or replies."`
	Limit      *int   `json:"limit,omitempty"        jsonschema:"description=Maximum turns to return (default 10, max 50)."`
	MaxCellLen *int   `json:"max_cell_len,omitempty" jsonschema:"description=Maximum rendered characters (default 200, max 10000)."`
}

type Output struct {
	History string `json:"history"`
}

// Tool returns the history tool, backed exclusively by the Store's read-only
// handle. A nil store leaves the tool unconfigured and it fails on use.
func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, d.Store, in)
	})
}

func (in Input) Validate() error {
	if in.Limit != nil && (*in.Limit < 1 || *in.Limit > maxLimit) {
		return fmt.Errorf("limit must be between 1 and %d", maxLimit)
	}
	if in.MaxCellLen != nil && (*in.MaxCellLen < 1 || *in.MaxCellLen > maxCellLen) {
		return fmt.Errorf("max_cell_len must be between 1 and %d", maxCellLen)
	}
	return nil
}
