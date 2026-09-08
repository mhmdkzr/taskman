// Package review wires up `taskman task review <command>` - the review
// stage's automated (record) and human (approve/reject) commands.
package review

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/task/review/approve"
	"github.com/mhmdkzr/taskman/internal/cli/task/review/record"
	"github.com/mhmdkzr/taskman/internal/cli/task/review/reject"
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
