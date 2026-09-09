package update

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "update" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "patch a task's title, labels, references, trunk, or auto-approve setting",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title", Usage: "new title"},
			&cli.StringSliceFlag{Name: "label", Usage: "set a label as key=value - repeatable"},
			&cli.StringSliceFlag{Name: "unset-label", Usage: "remove a label by key - repeatable"},
			&cli.StringSliceFlag{Name: "reference", Usage: "replace the reference list - repeatable"},
			&cli.BoolFlag{Name: "clear-references", Usage: "remove every reference"},
			&cli.BoolFlag{
				Name:  "trunk",
				Usage: "work this task on the current branch instead of an isolated worktree/branch (pass --trunk=false to unset)",
			},
			&cli.BoolFlag{
				Name:  "auto-approve",
				Usage: "skip the human review gate - review completes on its own once the commit is made (pass --auto-approve=false to unset)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			labels, err := utils.SplitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			req := Request{
				ID:              id,
				SetLabels:       labels,
				UnsetLabels:     cmd.StringSlice("unset-label"),
				References:      cmd.StringSlice("reference"),
				ClearReferences: cmd.Bool("clear-references"),
			}
			if cmd.IsSet("title") {
				title := cmd.String("title")
				req.Title = &title
			}
			if cmd.IsSet("trunk") {
				trunk := cmd.Bool("trunk")
				req.Trunk = &trunk
			}
			if cmd.IsSet("auto-approve") {
				autoApprove := cmd.Bool("auto-approve")
				req.AutoApprove = &autoApprove
			}
			t, err := Update(cmd.String("tasks-dir"), req)
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
