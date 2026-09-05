package reset

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

func execute(_ context.Context, c *browser.Client) (output, error) {
	result, err := c.Reset()
	if err != nil {
		return output{}, fmt.Errorf("browser_reset: %w", err)
	}
	return output{Reset: result.Reset}, nil
}
