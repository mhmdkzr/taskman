package task

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
	"github.com/mhmdkzr/loop/internal/task"
)

// Merge returns the "merge" command.
func Merge() *cli.Command {
	return &cli.Command{
		Name:      "merge",
		Usage:     "record that you already merged the task's branch",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "commit",
				Usage: "override the recorded commit hash, e.g. after a non-fast-forward merge",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := task.Merge(support.RepoFrom(cmd), id, task.MergeRequest{Commit: cmd.String("commit")})
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
