// Package human groups "specification review human approved" and
// "specification review human rejected". It has no behavior of its own.
package human

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/specification/review/human/approved"
	"github.com/mhmdkzr/taskman/internal/commands/specification/review/human/rejected"
)

// Command returns the "human" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "human",
		Usage: "a specification's human review",
		Commands: []*cli.Command{
			approved.Command(),
			rejected.Command(),
		},
	}
}
