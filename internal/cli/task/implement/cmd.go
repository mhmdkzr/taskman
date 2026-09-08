package implement

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
)

// Command returns the "implement" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "implement",
		Usage:     "mark a task's implementation attempt as done",
		ArgsUsage: "<id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Implement(cmd.String("tasks-dir"), id)
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
