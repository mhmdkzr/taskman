package task

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
	"github.com/mhmdkzr/loop/internal/task"
)

// Verify returns the "verify" command.
func Verify() *cli.Command {
	return &cli.Command{
		Name:      "verify",
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
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			checks, err := support.ParseChecks(cmd.StringSlice("check"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			t, err := task.Verify(support.RepoFrom(cmd), id, task.VerifyRequest{
				Checks: checks,
				Output: cmd.String("output"),
			})
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
