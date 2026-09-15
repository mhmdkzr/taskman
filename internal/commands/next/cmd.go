package next

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	taskview "github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "next" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "next",
		Usage: "show what should happen next for a task, and how to report it",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task id to inspect"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			st, err := store.Open(ctx, cmd.String("db"))
			if err != nil {
				return utils.Fail(err)
			}
			defer func() {
				if err := st.Close(); err != nil {
					slog.Error("close store", "error", err)
				}
			}()

			guidance, err := Next(ctx, st, Request{ID: id})
			if err != nil {
				return utils.Fail(err)
			}
			return printGuidance(cmd, guidance)
		},
	}
}

func printGuidance(cmd *cli.Command, guidance Instructions) error {
	if cmd.Bool("json") {
		if err := taskview.PrintJSON(cmd, guidance); err != nil {
			return fmt.Errorf("print guidance: %w", err)
		}
		return nil
	}
	out := cmd.Root().Writer
	if _, err := fmt.Fprintf(out, "%s (%s)\n\n%s\n", guidance.Action, guidance.State, guidance.Message); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if len(guidance.Commands) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(out, "\nReport with:"); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	for _, command := range guidance.Commands {
		if _, err := fmt.Fprintf(out, "  taskman %s\n", command); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}
	return nil
}
