package merge

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
)

// Command returns the "merge" command.
func Command() *cli.Command {
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
			t, err := Merge(cmd.String("tasks-dir"), id, Request{Commit: cmd.String("commit")})
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
