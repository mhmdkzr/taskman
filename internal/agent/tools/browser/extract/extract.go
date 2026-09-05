package extract

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

//nolint:contextcheck // Client operations use the browser's context-bound Rod session.
func execute(_ context.Context, c *browser.Client, in input) (output, error) {
	result, err := c.Extract(in.Selector)
	if err != nil {
		return output{}, fmt.Errorf("browser_extract: %w", err)
	}
	return output{URL: result.URL, Title: result.Title, Selector: result.Selector, Text: result.Text}, nil
}
