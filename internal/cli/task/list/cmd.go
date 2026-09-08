package list

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
	"github.com/mhmdkzr/taskman/internal/task"
)

// defaultLimit caps a call's page size when --limit isn't given, so
// `task list` against a large tasks-dir doesn't dump everything at once.
const defaultLimit = 50

// Result is task list's JSON output shape: the page of tasks plus enough to
// tell whether more pages remain.
type Result struct {
	Tasks  []task.Task `json:"tasks"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

// Command returns the "list" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list tasks, optionally filtered and paginated",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "state",
				Usage: "filter by task state (created, started, blocked, completed, failed) - repeatable",
			},
			&cli.StringSliceFlag{Name: "label", Usage: "filter by label as key=value - repeatable"},
			&cli.IntFlag{
				Name:  "limit",
				Value: defaultLimit,
				Usage: "max tasks to return; 0 for unlimited",
			},
			&cli.IntFlag{Name: "offset", Usage: "skip this many matching tasks before the page starts"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			stateFlags := cmd.StringSlice("state")
			states := make([]task.State, len(stateFlags))
			for i, s := range stateFlags {
				states[i] = task.State(s)
			}
			labels, err := support.SplitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			limit := cmd.Int("limit")
			offset := cmd.Int("offset")
			filter := Filter{State: states, Labels: labels, Limit: limit, Offset: offset}
			tasks, total, err := List(cmd.String("tasks-dir"), filter)
			if err != nil {
				return support.Fail(err)
			}
			if cmd.Bool("json") {
				return support.PrintJSON(cmd, Result{Tasks: tasks, Total: total, Limit: limit, Offset: offset})
			}
			if len(tasks) == 0 {
				if _, err := fmt.Fprintln(cmd.Root().Writer, "no tasks"); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
				return nil
			}
			for _, t := range tasks {
				if err := support.PrintTask(cmd, t); err != nil {
					return fmt.Errorf("print task: %w", err)
				}
			}
			if shown := offset + len(tasks); shown < total {
				if _, err := fmt.Fprintf(
					cmd.Root().Writer, "... %d more (use --offset %d to see the rest)\n", total-shown, shown,
				); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
			}
			return nil
		},
	}
}
