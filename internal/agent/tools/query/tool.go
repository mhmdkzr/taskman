package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "query"
	Description = "Run a read-only SQL query against the agent database and return the results as a table."
)

type Input struct {
	Query      string `json:"query"                  jsonschema:"description=Read-only SQL SELECT statement."`
	MaxCellLen *int   `json:"max_cell_len,omitempty" jsonschema:"description=Maximum characters rendered per cell (default 200, max 10000)."`
}

type Output struct {
	Table string `json:"table"`
}

// Tool returns a query tool backed exclusively by the Store's read-only
// handle. A nil store leaves the tool unconfigured and it fails on use.
func Tool(d tools.Deps) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return Execute(ctx, d.Store, in)
	})
}

func (in Input) Validate() error {
	if strings.TrimSpace(in.Query) == "" {
		return fmt.Errorf("query is required")
	}
	if in.MaxCellLen != nil && (*in.MaxCellLen < 1 || *in.MaxCellLen > maxCellLen) {
		return fmt.Errorf("max_cell_len must be between 1 and %d", maxCellLen)
	}
	return nil
}
