package escalate

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "escalated" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "escalated",
		Usage:     "block a task because a dispatched agent gave up",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "stage",
				Required: true,
				Usage:    "which stage was in flight (definition, specification, implementation, verification, review, or merge)",
			},
			&cli.StringFlag{Name: "reason", Required: true, Usage: "why the agent gave up"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Escalate(cmd.String("tasks-dir"), Request{
				ID:     id,
				Stage:  cmd.String("stage"),
				Reason: cmd.String("reason"),
			})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
