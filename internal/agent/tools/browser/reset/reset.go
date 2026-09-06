package reset

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

func execute(_ context.Context, c *browser.Client) (Output, error) {
	result, err := c.Reset()
	if err != nil {
		return Output{}, fmt.Errorf("browser_reset: %w", err)
	}
	return Output{Reset: result.Reset}, nil
}
