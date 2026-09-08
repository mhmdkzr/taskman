package commit

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
)

// Command returns the "commit" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "commit",
		Usage:     "read back the commit you already made",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "commit", Usage: "read this commit-ish instead of HEAD"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Commit(ctx, cmd.String("tasks-dir"), support.GitFrom(cmd), id, Request{
				Commit: cmd.String("commit"),
			})
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
