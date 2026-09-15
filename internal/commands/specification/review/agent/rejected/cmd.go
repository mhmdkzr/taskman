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
		Usage: "report a task's specification's automated review as rejected",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Required: true,
				Usage:    "the task whose specification's automated review was rejected",
			},
			&cli.StringSliceFlag{Name: "finding", Required: true, Usage: "a finding as location=detail - repeatable"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			findings, err := utils.ParseFindings(cmd.StringSlice("finding"))
			if err != nil {
				return cli.Exit(err, 2)
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

			t, err := Rejected(ctx, st, Request{ID: id, Findings: findings})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
