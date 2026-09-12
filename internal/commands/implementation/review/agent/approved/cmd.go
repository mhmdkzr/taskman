package approved

import (
	"context"

	"github.com/urfave/cli/v3"

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
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer utils.CloseStore(st)

			t, err := Approved(ctx, st, Request{ID: id, Comment: cmd.String("comment")})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
