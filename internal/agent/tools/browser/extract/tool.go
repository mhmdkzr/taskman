// Package extract implements browser text extraction.
package extract

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

const (
	Name        = "browser_extract"
	description = "Extract visible text from the current browser page or a specific element. Call browser_navigate first to open a page."
)

const Description = description

type input struct {
	Selector string `json:"selector,omitempty" jsonschema:"description=CSS selector of the element to extract text from. When empty, extracts visible text from the whole page body."`
}

type output struct {
	URL      string `json:"url"`
	Title    string `json:"title,omitempty"`
	Selector string `json:"selector,omitempty"`
	Text     string `json:"text"`
}

type Input = input
type Output = output

// Tool returns the browser_extract tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, in input) (output, error) {
		return execute(ctx, c, in)
	})
}

func (input) Validate() error { return nil }
