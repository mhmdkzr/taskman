package task

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
	"github.com/mhmdkzr/loop/internal/task"
)

// Create returns the "create" command.
func Create() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "create a task and its worktree/branch",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "definition", Required: true, Usage: "what the task should accomplish"},
			&cli.StringFlag{Name: "id", Usage: "use this id instead of generating one"},
			&cli.StringFlag{Name: "title", Usage: "short human-readable title"},
			&cli.StringSliceFlag{Name: "label", Usage: "a label as key=value - repeatable"},
			&cli.StringSliceFlag{Name: "reference", Usage: "a file or location relevant to this task - repeatable"},
			&cli.StringFlag{
				Name:  "specification",
				Usage: "skip straight to implementation by providing the specification up front (requires --done-when)",
			},
			&cli.StringFlag{Name: "done-when", Usage: "acceptance criteria - required together with --specification"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			labels, err := support.SplitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			t, err := task.Create(
				ctx,
				support.RepoFrom(cmd),
				support.GitFrom(cmd),
				support.WorktreesDirFrom(cmd),
				task.CreateRequest{
					Definition:    cmd.String("definition"),
					ID:            cmd.String("id"),
					Title:         cmd.String("title"),
					Labels:        labels,
					References:    cmd.StringSlice("reference"),
					Specification: cmd.String("specification"),
					DoneWhen:      cmd.String("done-when"),
				},
			)
			if err != nil {
				return support.Fail(err)
			}
			if cmd.Bool("json") {
				return support.PrintJSON(cmd, t)
			}
			if _, err := fmt.Fprintln(cmd.Root().Writer, support.CreateSummary(t)); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}
