package delete

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "delete" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "permanently delete a task and its event log",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task id to delete"},
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

			result, err := Delete(ctx, st, Request{ID: id})
			if err != nil {
				return utils.Fail(err)
			}

			// There is no surviving task to render, so delete has its own
			// three-way output instead of utils.PrintTask.
			switch {
			case cmd.Bool("json"):
				return utils.PrintJSON(cmd, result)
			case cmd.Bool("md"):
				_, err = fmt.Fprintf(cmd.Root().Writer, "# Deleted task %s\n", result.ID)
			default:
				_, err = fmt.Fprintf(cmd.Root().Writer, "deleted task %s\n", result.ID)
			}
			if err != nil {
				return utils.Fail(err)
			}
			return nil
		},
	}
}
