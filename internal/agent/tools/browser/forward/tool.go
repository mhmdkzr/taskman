// Package forward implements browser history forward navigation.
package forward

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

const (
	Name        = "browser_forward"
	description = "Navigate one step forward in the browser history."
)

const Description = description

type input struct{}

type output struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

type Input = input
type Output = output

// Tool returns the browser_forward tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, _ input) (output, error) {
		return execute(ctx, c)
	})
}

func (input) Validate() error { return nil }
