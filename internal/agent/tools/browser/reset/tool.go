// Package reset implements browser page reset.
package reset

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

const (
	Name        = "browser_reset"
	description = "Close the current browser page so the next navigation starts fresh. Keeps the browser process alive."
)

type input struct{}

type output struct {
	Reset bool `json:"reset"`
}

// Tool returns the browser_reset tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, description, func(ctx context.Context, _ input) (output, error) {
		return execute(ctx, c)
	})
}

func (input) Validate() error { return nil }
