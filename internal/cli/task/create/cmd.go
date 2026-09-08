package create

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
)

// Command returns the "create" command.
func Command() *cli.Command {
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
			t, err := Create(
				ctx,
				cmd.String("tasks-dir"),
				support.WorktreesDirFrom(cmd),
				support.GitFrom(cmd),
				Request{
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
			if _, err := fmt.Fprintln(cmd.Root().Writer, Summary{
				TaskID:   t.ID,
				Title:    t.Title,
				Worktree: t.Git.Worktree,
				Branch:   t.Git.Branch,
			}.Render()); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}
