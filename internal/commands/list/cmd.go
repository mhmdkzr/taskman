package list

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/view/json"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "list" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list every task",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer utils.CloseStore(st)

			tasks, err := List(ctx, st)
			if err != nil {
				return utils.Fail(err)
			}
			if cmd.Bool("json") {
				docs := make([]json.Document, 0, len(tasks))
				for _, t := range tasks {
					docs = append(docs, json.FromTask(t))
				}
				return utils.PrintJSON(cmd, docs)
			}
			if len(tasks) == 0 {
				_, err := fmt.Fprintln(cmd.Root().Writer, "no tasks")
				return utils.Fail(err)
			}
			if cmd.Bool("md") {
				for i, t := range tasks {
					if i > 0 {
						if _, err := fmt.Fprintln(cmd.Root().Writer, "\n---"); err != nil {
							return utils.Fail(err)
						}
					}
					if err := utils.PrintMarkdown(cmd, t); err != nil {
						return utils.Fail(err)
					}
				}
				return nil
			}
			for _, t := range tasks {
				if err := utils.PrintTask(cmd, t); err != nil {
					return utils.Fail(err)
				}
			}
			return nil
		},
	}
}
