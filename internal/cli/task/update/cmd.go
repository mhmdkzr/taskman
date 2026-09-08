package update

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
)

// Command returns the "update" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "patch a task's title, labels, or references",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title", Usage: "new title"},
			&cli.StringSliceFlag{Name: "label", Usage: "set a label as key=value - repeatable"},
			&cli.StringSliceFlag{Name: "unset-label", Usage: "remove a label by key - repeatable"},
			&cli.StringSliceFlag{Name: "reference", Usage: "replace the reference list - repeatable"},
			&cli.BoolFlag{Name: "clear-references", Usage: "remove every reference"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			labels, err := support.SplitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			req := Request{
				SetLabels:       labels,
				UnsetLabels:     cmd.StringSlice("unset-label"),
				References:      cmd.StringSlice("reference"),
				ClearReferences: cmd.Bool("clear-references"),
			}
			if cmd.IsSet("title") {
				title := cmd.String("title")
				req.Title = &title
			}
			t, err := Update(cmd.String("tasks-dir"), id, req)
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
