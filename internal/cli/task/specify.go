//nolint:dupl // thin per-command CLI wiring - shares a body shape with escalate.go but a distinct flag surface
package task

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
	"github.com/mhmdkzr/loop/internal/task"
)

// Specify returns the "specify" command.
func Specify() *cli.Command {
	return &cli.Command{
		Name:      "specify",
		Usage:     "write a task's specification and acceptance criteria",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "result", Required: true, Usage: "the drafted specification"},
			&cli.StringFlag{Name: "done-when", Required: true, Usage: "the drafted acceptance criteria"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := task.Specify(support.RepoFrom(cmd), id, task.SpecifyRequest{
				Result:   cmd.String("result"),
				DoneWhen: cmd.String("done-when"),
			})
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
