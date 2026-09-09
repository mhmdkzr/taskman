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
		Usage: "delete every completed task file - a human housekeeping action",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "list the completed tasks that would be deleted without deleting them",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			res, err := Prune(cmd.String("tasks-dir"), Request{DryRun: cmd.Bool("dry-run")})
			if err != nil {
				return utils.Fail(err)
			}
			if cmd.Bool("json") {
				return utils.PrintJSON(cmd, res)
			}
			if res.Count == 0 {
				if _, err := fmt.Fprintln(cmd.Root().Writer, "no completed tasks"); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
				return nil
			}
			verb := "deleted"
			if cmd.Bool("dry-run") {
				verb = "would delete"
			}
			for _, id := range res.Deleted {
				if _, err := fmt.Fprintf(cmd.Root().Writer, "%s %s\n", verb, id); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
			}
			if _, err := fmt.Fprintf(cmd.Root().Writer, "%d completed task(s)\n", res.Count); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}
