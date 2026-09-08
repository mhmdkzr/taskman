// Package review wires up `taskman review <command>` - the review stage's
// automated (record) and human (approve/reject) commands.
package review

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/review/approve"
	"github.com/mhmdkzr/taskman/internal/commands/review/record"
	"github.com/mhmdkzr/taskman/internal/commands/review/reject"
)

// Command builds the "review" command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "review",
		Usage: "the review stage: automated (record) and human (approve/reject)",
		Commands: []*cli.Command{
			record.Command(),
			approve.Command(),
			reject.Command(),
		},
	}
}
