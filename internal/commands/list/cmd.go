package list

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "list" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list every task",
		Action: func(_ context.Context, cmd *cli.Command) error {
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer st.Close()

			tasks, err := List(st)
			if err != nil {
				return utils.Fail(err)
			}
			if cmd.Bool("json") {
				return utils.PrintJSON(cmd, tasks)
			}
			if len(tasks) == 0 {
				_, err := fmt.Fprintln(cmd.Root().Writer, "no tasks")
				return err
			}
			for _, t := range tasks {
				if err := utils.PrintTask(cmd, t); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
