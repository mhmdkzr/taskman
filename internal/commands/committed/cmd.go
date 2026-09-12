package committed

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "committed" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "committed",
		Usage: "record the task's implementation worktree's current commit",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task whose commit was recorded"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return err
			}
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer st.Close()

			t, err := Committed(ctx, st, utils.GitFrom(cmd), Request{ID: id})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
