package escalated

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "escalated" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "escalated",
		Usage: "block a task pending outside intervention",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "the task being escalated"},
			&cli.StringFlag{Name: "stage", Usage: "the stage the task is stuck at"},
			&cli.StringFlag{Name: "reason", Usage: "why the task is stuck"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			if err := utils.RequireFlags(cmd, "stage", "reason"); err != nil {
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

			t, err := Escalated(ctx, st, Request{ID: id, Stage: cmd.String("stage"), Reason: cmd.String("reason")})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
