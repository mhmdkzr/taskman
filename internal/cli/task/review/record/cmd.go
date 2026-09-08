package record

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
)

// Command returns the "record" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "record",
		Usage:     "report the automated review round's verdict",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:     "approved",
				Required: true,
				Usage:    "whether the automated review approved this attempt",
			},
			&cli.StringSliceFlag{Name: "finding", Usage: "a finding as file=detail - repeatable"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			findings, err := support.ParseFindings(cmd.StringSlice("finding"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			t, err := RecordReview(cmd.String("tasks-dir"), id, Request{
				Approved: cmd.Bool("approved"),
				Findings: findings,
			})
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
