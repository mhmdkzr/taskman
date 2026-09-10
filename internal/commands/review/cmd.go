// Package review wires up the human review gate.
package review

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/review/approve"
	"github.com/mhmdkzr/taskman/internal/commands/review/reject"
)

// Command builds the "review" command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "review",
		Usage: "the human review gate",
		Commands: []*cli.Command{
			approve.Command(),
			reject.Command(),
		},
	}
}
