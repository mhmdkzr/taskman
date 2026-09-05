package scroll

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

//nolint:contextcheck // Client operations use the browser's context-bound Rod session.
func execute(_ context.Context, c *browser.Client, in input) (output, error) {
	result, err := c.Scroll(in.Direction, in.Selector)
	if err != nil {
		return output{}, fmt.Errorf("browser_scroll: %w", err)
	}
	return output{Scrolled: result.Scrolled, Direction: result.Direction, Selector: result.Selector}, nil
}
