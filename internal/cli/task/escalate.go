//nolint:dupl // thin per-command CLI wiring - shares a body shape with specify.go but a distinct flag surface
package task

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
	"github.com/mhmdkzr/loop/internal/task"
)

// Escalate returns the "escalate" command.
func Escalate() *cli.Command {
	return &cli.Command{
		Name:      "escalate",
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
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := task.Escalate(support.RepoFrom(cmd), id, task.EscalateRequest{
				Stage:  cmd.String("stage"),
				Reason: cmd.String("reason"),
			})
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
