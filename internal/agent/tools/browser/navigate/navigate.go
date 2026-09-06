package navigate

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

//nolint:contextcheck // Client operations use the browser's context-bound Rod session.
func execute(_ context.Context, c *browser.Client, in Input) (Output, error) {
	result, err := c.Navigate(in.URL)
	if err != nil {
		return Output{}, fmt.Errorf("browser_navigate: %w", err)
	}
	return Output{URL: result.URL, Title: result.Title}, nil
}
