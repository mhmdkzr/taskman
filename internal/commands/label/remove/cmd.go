package remove

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "label remove" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "remove",
		Usage: "delete one or more of a task's labels",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "the task whose labels are being removed"},
			&cli.StringSliceFlag{Name: "key", Usage: "a label key to delete - repeatable"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			if err := utils.RequireFlags(cmd, "key"); err != nil {
				return err
			}
			st, err := store.Open(ctx, cmd.String("db"))
			if err != nil {
				return utils.Fail(err)
			}
			defer func() {
				if err := st.Close(); err != nil {
					slog.Error("close store", "error", err)
				}
			}()

			t, err := Remove(ctx, st, Request{ID: id, Keys: cmd.StringSlice("key")})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
