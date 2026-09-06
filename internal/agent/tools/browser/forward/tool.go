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
	Description = "Navigate one step forward in the browser history."
)

type Input struct{}

type Output struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// Tool returns the browser_forward tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, _ Input) (Output, error) {
		return execute(ctx, c)
	})
}

func (Input) Validate() error { return nil }
