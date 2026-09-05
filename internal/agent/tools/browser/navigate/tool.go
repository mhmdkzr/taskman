// Package navigate implements browser page navigation.
package navigate

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

const (
	Name        = "browser_navigate"
	description = "Navigate the browser to a URL and wait for the page to load. Use it to open a website before extracting content or interacting with it."
)

type input struct {
	URL string `json:"url" jsonschema:"description=The URL to navigate to, e.g. https://example.com"`
}

type output struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// Tool returns the browser_navigate tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, in input) (output, error) {
		return execute(ctx, c, in)
	})
}

func (input) Validate() error { return nil }
