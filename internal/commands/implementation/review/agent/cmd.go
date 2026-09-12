// Package agent groups "implementation review agent approved" and
// "implementation review agent rejected". It has no behavior of its own.
package agent

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/implementation/review/agent/approved"
	"github.com/mhmdkzr/taskman/internal/commands/implementation/review/agent/rejected"
)

// Command returns the "agent" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "agent",
		Usage: "an implementation's automated review",
		Commands: []*cli.Command{
			approved.Command(),
			rejected.Command(),
		},
	}
}
