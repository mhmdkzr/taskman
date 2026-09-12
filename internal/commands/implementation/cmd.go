// Package implementation groups a task's implementation-stage review
// actions under "implementation review agent ..." and "implementation
// review human ...". It has no behavior of its own - "implemented" (the
// ImplementationCompleted report) stays a separate top-level command.
package implementation

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/implementation/review"
)

// Command returns the "implementation" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "implementation",
		Usage: "a task's implementation-stage review",
		Commands: []*cli.Command{
			review.Command(),
		},
	}
}
