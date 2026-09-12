package create

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "create" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "create a new task",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title", Usage: "short human-readable title"},
			&cli.StringFlag{Name: "description", Required: true, Usage: "what the task should accomplish"},
			&cli.StringSliceFlag{Name: "label", Usage: "a label as key=value - repeatable"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			labels, err := utils.SplitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer st.Close()

			t, err := Create(ctx, st, Request{
				Title:       cmd.String("title"),
				Description: cmd.String("description"),
				Labels:      labels,
			})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
