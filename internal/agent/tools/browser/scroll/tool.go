// Package scroll implements browser page scrolling.
package scroll

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

const (
	Name        = "browser_scroll"
	description = "Scroll the current browser page or scroll an element into view."
)

const Description = description

type input struct {
	Direction string `json:"direction,omitempty" jsonschema:"description=Scroll direction when no selector is given: up, down, top, or bottom. Defaults to down."`
	Selector  string `json:"selector,omitempty"  jsonschema:"description=CSS selector of the element to scroll into view. When set, direction is ignored."`
}

type output struct {
	Scrolled  bool   `json:"scrolled"`
	Direction string `json:"direction,omitempty"`
	Selector  string `json:"selector,omitempty"`
}

type Input = input
type Output = output

// Tool returns the browser_scroll tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, in input) (output, error) {
		return execute(ctx, c, in)
	})
}

func (input) Validate() error { return nil }
