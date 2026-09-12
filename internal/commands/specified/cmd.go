package specified

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "specified" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "specified",
		Usage: "report a task's specification as submitted",
		Flags: append([]cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task whose specification was submitted"},
			&cli.StringFlag{Name: "plan", Required: true, Usage: "the specification's plan"},
		}, utils.ReviewFlags()...),
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

			t, err := Specified(ctx, st, Request{
				ID:     id,
				Plan:   cmd.String("plan"),
				Review: utils.ReviewConfigurationFrom(cmd),
			})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
