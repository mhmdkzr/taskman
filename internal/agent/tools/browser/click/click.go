package click

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

//nolint:contextcheck // Client operations use the browser's context-bound Rod session.
func execute(_ context.Context, c *browser.Client, in Input) (Output, error) {
	result, err := c.Click(in.Selector)
	if err != nil {
		return Output{}, fmt.Errorf("browser_click: %w", err)
	}
	return Output{Clicked: result.Clicked, URL: result.URL, Title: result.Title}, nil
}
