package search

import (
	"context"
	"fmt"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/zendev-sh/goai"
	"strings"
)

const Name = "notes_search"
const Description = "Search persistent note names and bodies for the current session."

type Input struct {
	Query string `json:"query" jsonschema:"description=FTS search query."`
	Limit *int   `json:"limit,omitempty" jsonschema:"description=Maximum notes to return (default 20, max 100)."`
}
type Note struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}
type Output struct {
	Notes []Note `json:"notes"`
	Limit int    `json:"limit"`
}

func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) { return execute(ctx, d, in) })
}
func (in Input) Validate() error {
	if strings.TrimSpace(in.Query) == "" {
		return fmt.Errorf("query is required")
	}
	if in.Limit != nil && (*in.Limit < 1 || *in.Limit > 100) {
		return fmt.Errorf("limit must be between 1 and 100")
	}
	return nil
}
