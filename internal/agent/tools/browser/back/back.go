package back

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

//nolint:contextcheck // Client operations use the browser's context-bound Rod session.
func execute(_ context.Context, c *browser.Client) (output, error) {
	result, err := c.Back()
	if err != nil {
		return output{}, fmt.Errorf("browser_back: %w", err)
	}
	return output{URL: result.URL, Title: result.Title}, nil
}
