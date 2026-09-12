package abandoned

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "abandoned" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "abandoned",
		Usage: "end a task unsuccessfully",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task being abandoned"},
			&cli.StringFlag{Name: "reason", Required: true, Usage: "why the task is being abandoned"},
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

			t, err := Abandoned(ctx, st, Request{ID: id, Reason: cmd.String("reason")})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
