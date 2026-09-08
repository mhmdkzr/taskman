package list

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
	"github.com/mhmdkzr/taskman/internal/task"
)

// Command returns the "list" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list tasks, optionally filtered",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "state",
				Usage: "filter by task state (created, started, blocked, completed, failed) - repeatable",
			},
			&cli.StringSliceFlag{Name: "label", Usage: "filter by label as key=value - repeatable"},
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
			tasks, err := List(cmd.String("tasks-dir"), Filter{State: states, Labels: labels})
			if err != nil {
				return support.Fail(err)
			}
			if cmd.Bool("json") {
				return support.PrintJSON(cmd, tasks)
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
			return nil
		},
	}
}
