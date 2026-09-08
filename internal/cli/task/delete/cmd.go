package delete

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
)

// Command returns the "delete" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "remove a task file outright - a human housekeeping action",
		ArgsUsage: "<id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			if err := Delete(cmd.String("tasks-dir"), id); err != nil {
				return support.Fail(err)
			}
			if _, err := fmt.Fprintf(cmd.Root().Writer, "deleted task %s\n", id); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}
