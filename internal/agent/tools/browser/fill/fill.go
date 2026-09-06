package fill

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

//nolint:contextcheck // Client operations use the browser's context-bound Rod session.
func execute(_ context.Context, c *browser.Client, in Input) (Output, error) {
	result, err := c.Fill(in.Selector, in.Text)
	if err != nil {
		return Output{}, fmt.Errorf("browser_fill: %w", err)
	}
	return Output{Filled: result.Filled, Text: result.Text}, nil
}
