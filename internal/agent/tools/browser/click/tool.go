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
	description = "Click an element on the current browser page by CSS selector. The page must already be open via browser_navigate."
)

type input struct {
	Selector string `json:"selector" jsonschema:"description=CSS selector of the element to click, e.g. \"button.submit\" or \"#login\""`
}

type output struct {
	Clicked string `json:"clicked"`
	URL     string `json:"url"`
	Title   string `json:"title,omitempty"`
}

// Tool returns the browser_click tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, in input) (output, error) {
		return execute(ctx, c, in)
	})
}

func (input) Validate() error { return nil }
