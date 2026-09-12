package rejected

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "rejected" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "rejected",
		Usage: "report a task's specification's human review as rejected",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task whose specification's human review was rejected"},
			&cli.StringFlag{Name: "reason", Required: true, Usage: "why the specification's human review was rejected"},
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

			t, err := Rejected(ctx, st, Request{ID: id, Reason: cmd.String("reason")})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
