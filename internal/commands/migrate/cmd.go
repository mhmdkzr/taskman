package migrate

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "migrate legacy task files to the current schema",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "dry-run", Usage: "validate and show migrations without rewriting task files"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			result, err := Migrate(cmd.String("tasks-dir"), Request{DryRun: cmd.Bool("dry-run")})
			if err != nil {
				return utils.Fail(err)
			}
			if cmd.Bool("json") {
				return utils.PrintJSON(cmd, result)
			}
			verb := "migrated"
			if cmd.Bool("dry-run") {
				verb = "would migrate"
			}
			for _, change := range result.Migrated {
				if _, err := fmt.Fprintf(
					cmd.Root().Writer,
					"%s %s: %s -> %s\n",
					verb,
					change.ID,
					change.From,
					change.To,
				); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
			}
			if _, err := fmt.Fprintf(
				cmd.Root().Writer,
				"%d task(s) %s; %d already current\n",
				len(result.Migrated),
				verb,
				len(result.Skipped),
			); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}
}
