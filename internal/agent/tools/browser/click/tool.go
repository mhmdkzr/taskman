// Package click implements browser element clicking.
package click

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

const (
	Name        = "browser_click"
	Description = "Click an element on the current browser page by CSS selector. The page must already be open via browser_navigate."
)

type Input struct {
	Selector string `json:"selector" jsonschema:"description=CSS selector of the element to click, e.g. \"button.submit\" or \"#login\""`
}

type Output struct {
	Clicked string `json:"clicked"`
	URL     string `json:"url"`
	Title   string `json:"title,omitempty"`
}

// Tool returns the browser_click tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, in Input) (Output, error) {
		return execute(ctx, c, in)
	})
}

func (Input) Validate() error { return nil }
