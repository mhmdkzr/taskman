package next

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "next" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "next",
		Usage:     "show what should happen next for this task",
		ArgsUsage: "<id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			g, err := Next(cmd.String("tasks-dir"), Request{ID: id})
			if err != nil {
				return utils.Fail(err)
			}
			if cmd.Bool("json") {
				return utils.PrintJSON(cmd, g)
			}
			if _, err := fmt.Fprintln(cmd.Root().Writer, g.Message); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}
