package abandon

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "abandon" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "abandon",
		Usage:     "mark a task failed for good",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "reason", Required: true, Usage: "why the task is being abandoned"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Abandon(
				ctx,
				cmd.String("tasks-dir"),
				utils.GitFrom(cmd),
				Request{ID: id, Reason: cmd.String("reason")},
			)
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
