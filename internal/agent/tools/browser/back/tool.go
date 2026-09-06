// Package back implements browser history back navigation.
package back

import (
	"context"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

const (
	Name        = "browser_back"
	Description = "Navigate one step back in the browser history."
)

type Input struct{}

type Output struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// Tool returns the web_browser_back tool bound to c.
func Tool(c *browser.Client) goai.Tool {
	return tools.Tool(Name, Description, func(ctx context.Context, _ Input) (Output, error) {
		return execute(ctx, c)
	})
}

func (Input) Validate() error { return nil }
