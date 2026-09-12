// Package specification groups a task's specification-stage review actions
// under "specification review agent ..." and "specification review human
// ...". It has no behavior of its own.
package specification

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/specification/review"
)

// Command returns the "specification" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "specification",
		Usage: "a task's specification-stage review",
		Commands: []*cli.Command{
			review.Command(),
		},
	}
}
