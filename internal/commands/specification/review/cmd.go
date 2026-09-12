// Package review groups a specification's agent and human review actions.
// It has no behavior of its own.
package review

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/specification/review/agent"
	"github.com/mhmdkzr/taskman/internal/commands/specification/review/human"
)

// Command returns the "review" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "review",
		Usage: "a specification's review",
		Commands: []*cli.Command{
			agent.Command(),
			human.Command(),
		},
	}
}
