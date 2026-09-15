package approved

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "approved" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "approved",
		Usage: "report a task's implementation's automated review as approved",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Required: true,
				Usage:    "the task whose implementation's automated review was approved",
			},
			&cli.StringFlag{Name: "comment", Usage: "an optional approval comment"},
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

			t, err := Approved(ctx, st, Request{ID: id, Comment: cmd.String("comment")})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
