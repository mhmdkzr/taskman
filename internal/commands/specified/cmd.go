package specified

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "specified" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "specified",
		Usage: "report a task's specification as submitted",
		Flags: append([]cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "the task whose specification was submitted"},
			&cli.StringFlag{Name: "plan", Usage: "the specification's plan"},
		}, utils.ReviewFlags()...),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			if err := utils.RequireFlags(cmd, "plan"); err != nil {
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

			t, err := Specified(ctx, st, Request{
				ID:     id,
				Plan:   cmd.String("plan"),
				Review: utils.ReviewConfigurationFrom(cmd),
			})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
