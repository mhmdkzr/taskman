package fill

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

//nolint:contextcheck // Client operations use the browser's context-bound Rod session.
func execute(_ context.Context, c *browser.Client, in input) (output, error) {
	result, err := c.Fill(in.Selector, in.Text)
	if err != nil {
		return output{}, fmt.Errorf("browser_fill: %w", err)
	}
	return output{Filled: result.Filled, Text: result.Text}, nil
}
