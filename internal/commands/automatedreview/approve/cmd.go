package approve

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "approved" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "approved",
		Usage:     "record an automated review approval",
		ArgsUsage: "<id>",
		Action: func(_ context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Approve(cmd.String("tasks-dir"), Request{ID: id})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
