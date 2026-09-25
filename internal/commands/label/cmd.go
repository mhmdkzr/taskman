// Package label groups a task's label-editing actions under "label add" and
// "label remove". It has no behavior of its own.
package label

import (
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/label/add"
	"github.com/mhmdkzr/taskman/internal/commands/label/remove"
)

// Command returns the "label" grouping command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "label",
		Usage: "add or remove a task's labels",
		Commands: []*cli.Command{
			add.Command(),
			remove.Command(),
		},
	}
}
