package prune

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "prune" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "prune",
		Usage: "permanently delete every completed task and its event log",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "report which tasks would be pruned without deleting them",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer utils.CloseStore(st)

			result, err := Prune(ctx, st, Request{DryRun: cmd.Bool("dry-run")})
			if err != nil {
				return utils.Fail(err)
			}

			// There is no surviving task to render, so prune has its own
			// three-way output instead of utils.PrintTask.
			switch {
			case cmd.Bool("json"):
				return utils.PrintJSON(cmd, result)
			case cmd.Bool("md"):
				err = renderMarkdown(cmd, result)
			default:
				err = renderHuman(cmd, result)
			}
			if err != nil {
				return utils.Fail(err)
			}
			return nil
		},
	}
}

func renderHuman(cmd *cli.Command, result Result) error {
	verb, noun := "pruned", "tasks"
	if result.DryRun {
		verb = "would prune"
	}
	if len(result.Pruned) == 1 {
		noun = "task"
	}
	if _, err := fmt.Fprintf(
		cmd.Root().Writer,
		"%s %d completed %s\n",
		verb,
		len(result.Pruned),
		noun,
	); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, id := range result.Pruned {
		if _, err := fmt.Fprintf(cmd.Root().Writer, "  %s\n", id); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	return nil
}

func renderMarkdown(cmd *cli.Command, result Result) error {
	heading := fmt.Sprintf("# Pruned %d completed tasks", len(result.Pruned))
	if result.DryRun {
		heading = fmt.Sprintf("# Would prune %d completed tasks (dry run)", len(result.Pruned))
	}
	if _, err := fmt.Fprintln(cmd.Root().Writer, heading); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, id := range result.Pruned {
		if _, err := fmt.Fprintf(cmd.Root().Writer, "- %s\n", id); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	return nil
}
