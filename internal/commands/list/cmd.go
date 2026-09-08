package list

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/utils"
)

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
			labels, err := utils.SplitKV(cmd.StringSlice("label"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			result, err := List(cmd.String("tasks-dir"), Request{
				State:  states,
				Labels: labels,
				Limit:  cmd.Int("limit"),
				Offset: cmd.Int("offset"),
			})
			if err != nil {
				return utils.Fail(err)
			}
			if cmd.Bool("json") {
				return utils.PrintJSON(cmd, result)
			}
			if len(result.Tasks) == 0 {
				if _, err := fmt.Fprintln(cmd.Root().Writer, "no tasks"); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
				return nil
			}
			for _, t := range result.Tasks {
				if err := utils.PrintTask(cmd, t); err != nil {
					return fmt.Errorf("print task: %w", err)
				}
			}
			if shown := result.Offset + len(result.Tasks); shown < result.Total {
				if _, err := fmt.Fprintf(
					cmd.Root().Writer, "... %d more (use --offset %d to see the rest)\n", result.Total-shown, shown,
				); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
			}
			return nil
		},
	}
}
