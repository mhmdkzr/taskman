// Package task wires up every `taskman task <command>` subcommand: parsing
// flags into a request, calling straight into internal/task's matching
// function, and rendering the result via internal/cli/support.
package task

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/task/review"
)

// Command builds the "task" command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "task",
		Usage: "create and drive tasks through their lifecycle",
		Commands: []*cli.Command{
			List(),
			Get(),
			Create(),
			Update(),
			Specify(),
			Implement(),
			Verify(),
			review.Command(),
			Commit(),
			Escalate(),
			Merge(),
			Abandon(),
			Next(),
			Delete(),
		},
	}
}
