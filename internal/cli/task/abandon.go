package task

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
	"github.com/mhmdkzr/loop/internal/task"
)

// Abandon returns the "abandon" command.
func Abandon() *cli.Command {
	return &cli.Command{
		Name:      "abandon",
		Usage:     "mark a task failed for good",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "reason", Required: true, Usage: "why the task is being abandoned"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := task.Abandon(support.RepoFrom(cmd), id, cmd.String("reason"))
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
