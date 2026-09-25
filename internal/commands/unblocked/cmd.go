package unblocked

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "unblocked" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "unblocked",
		Usage: "resume a blocked task, optionally granting more auto-fix rounds",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "the task being unblocked"},
			&cli.StringFlag{Name: "reason", Usage: "why the task is being unblocked"},
			&cli.IntFlag{Name: "rounds", Usage: "additional auto-fix rounds granted; required for a " +
				"budget-exhaustion blockage, must be omitted for an escalation"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			if err := utils.RequireFlags(cmd, "reason"); err != nil {
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

			t, err := Unblocked(ctx, st, Request{ID: id, Reason: cmd.String("reason"), Rounds: cmd.Int("rounds")})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
