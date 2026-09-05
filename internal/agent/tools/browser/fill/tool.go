// Package fill implements browser form filling.
package fill

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

const (
	Name        = "browser_fill"
	description = "Fill a form field on the current browser page. Focuses the element by CSS selector, clears it, and types the given text."
)

type input struct {
	Selector string `json:"selector" jsonschema:"description=CSS selector of the input element to fill, e.g. \"input[name=email]\" or \"#username\""`
	Text     string `json:"text"     jsonschema:"description=Text to type into the field"`
}

type output struct {
	Filled string `json:"filled"`
	Text   string `json:"text"`
}

// Tool returns the browser_fill tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, in input) (output, error) {
		return execute(ctx, c, in)
	})
}

func (input) Validate() error { return nil }
