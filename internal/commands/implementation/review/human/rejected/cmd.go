package rejected

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "rejected" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "rejected",
		Usage: "report a task's implementation's human review as rejected",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Required: true,
				Usage:    "the task whose implementation's human review was rejected",
			},
			&cli.StringFlag{
				Name:     "reason",
				Required: true,
				Usage:    "why the implementation's human review was rejected",
			},
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

			t, err := Rejected(ctx, st, Request{ID: id, Reason: cmd.String("reason")})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
