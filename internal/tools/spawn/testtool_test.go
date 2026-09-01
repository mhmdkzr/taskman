package spawn

import (
	"context"
	"time"

	"github.com/zendev-sh/goai"
)

// testClockTool returns a harmless tool named "datetime" that tests use as a
// base tool; it stands in for the production datetime tool, which the taskman
// tool set no longer includes.
func testClockTool() goai.Tool {
	return goai.NewTool("datetime", "returns the current UTC time",
		func(ctx context.Context, _ struct{}) (string, error) {
			return time.Now().UTC().Format(time.RFC3339), nil
		})
}
