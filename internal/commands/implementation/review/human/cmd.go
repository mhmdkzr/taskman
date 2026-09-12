// Package human groups "implementation review human approved" and
// "implementation review human rejected". It has no behavior of its own.
package human

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/implementation/review/human/approved"
	"github.com/mhmdkzr/taskman/internal/commands/implementation/review/human/rejected"
)

// Command returns the "human" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "human",
		Usage: "an implementation's human review",
		Commands: []*cli.Command{
			approved.Command(),
			rejected.Command(),
		},
	}
}
