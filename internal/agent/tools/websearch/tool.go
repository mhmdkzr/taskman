package websearch

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
)

const (
	Name        = "web_search"
	Description = "Search the web using Tavily and return a list of results with title, url and a short snippet for each."
)

type Input struct {
	Query      string `json:"query"                 jsonschema:"description=The search query."`
	MaxResults *int   `json:"max_results,omitempty" jsonschema:"description=Maximum number of results to return (default 5, max 10)."`
}

// Tool returns the web_search tool bound to c.
func Tool(c *Client) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, c, in)
	})
}

func (Input) Validate() error { return nil }
