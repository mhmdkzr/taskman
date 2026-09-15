package get

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "get" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "show a task's current state and instruction",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task id to read"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
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

			t, err := Get(ctx, st, Request{ID: id})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
