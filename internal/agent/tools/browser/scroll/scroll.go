package scroll

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/loop/internal/agent/tools/browser"
)

//nolint:contextcheck // Client operations use the browser's context-bound Rod session.
func execute(_ context.Context, c *browser.Client, in Input) (Output, error) {
	result, err := c.Scroll(in.Direction, in.Selector)
	if err != nil {
		return Output{}, fmt.Errorf("browser_scroll: %w", err)
	}
	return Output{Scrolled: result.Scrolled, Direction: result.Direction, Selector: result.Selector}, nil
}
