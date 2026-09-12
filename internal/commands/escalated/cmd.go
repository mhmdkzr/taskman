package escalated

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "escalated" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "escalated",
		Usage: "block a task pending outside intervention",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task being escalated"},
			&cli.StringFlag{Name: "stage", Required: true, Usage: "the stage the task is stuck at"},
			&cli.StringFlag{Name: "reason", Required: true, Usage: "why the task is stuck"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return err
			}
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer st.Close()

			t, err := Escalated(st, Request{ID: id, Stage: cmd.String("stage"), Reason: cmd.String("reason")})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
