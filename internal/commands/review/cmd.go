// Package review wires up `taskman review <command>` - the review stage's
// automated (recorded) and human (approved/rejected) commands.
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
		Usage: "the review stage: automated (recorded) and human (approved/rejected)",
		Commands: []*cli.Command{
			record.Command(),
			approve.Command(),
			reject.Command(),
		},
	}
}
