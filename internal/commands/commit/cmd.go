package commit

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
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
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Commit(ctx, cmd.String("tasks-dir"), utils.GitFrom(cmd), Request{
				ID:     id,
				Commit: cmd.String("commit"),
			})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
