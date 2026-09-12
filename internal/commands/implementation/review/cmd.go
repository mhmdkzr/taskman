// Package review groups an implementation's agent and human review
// actions. It has no behavior of its own.
package review

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/implementation/review/agent"
	"github.com/mhmdkzr/taskman/internal/commands/implementation/review/human"
)

// Command returns the "review" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "review",
		Usage: "an implementation's review",
		Commands: []*cli.Command{
			agent.Command(),
			human.Command(),
		},
	}
}
