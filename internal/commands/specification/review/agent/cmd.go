// Package agent groups "specification review agent approved" and
// "specification review agent rejected". It has no behavior of its own.
package agent

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/specification/review/agent/approved"
	"github.com/mhmdkzr/taskman/internal/commands/specification/review/agent/rejected"
)

// Command returns the "agent" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "agent",
		Usage: "a specification's automated review",
		Commands: []*cli.Command{
			approved.Command(),
			rejected.Command(),
		},
	}
}
