// Package automatedreview assembles the automated-review verdict commands.
package automatedreview

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/automatedreview/approve"
	"github.com/mhmdkzr/taskman/internal/commands/automatedreview/reject"
)

// Command builds the "automated-review" command tree.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "automated-review",
		Usage: "the automated review stage",
		Commands: []*cli.Command{
			approve.Command(),
			reject.Command(),
		},
	}
}
