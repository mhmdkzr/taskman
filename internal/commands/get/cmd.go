package get

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "get" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "show a task's current state and instruction",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task id to read"},
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

			t, err := Get(st, Request{ID: id})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
