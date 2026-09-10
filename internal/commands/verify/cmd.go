package verify

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "verified" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "verified",
		Usage:     "report one build-check attempt",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:     "check",
				Required: true,
				Usage:    "a build check result as name=ok|error, e.g. vet=ok - repeatable",
			},
			&cli.StringFlag{Name: "output", Usage: "combined output from the checks, for a human to read"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			checks, err := utils.ParseChecks(cmd.StringSlice("check"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			t, err := Verify(cmd.String("tasks-dir"), Request{
				ID:     id,
				Checks: checks,
				Output: cmd.String("output"),
			})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
