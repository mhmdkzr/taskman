package reject

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
)

// Command returns the "reject" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "reject",
		Usage:     "record a human's rejection and start review-reject recovery",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "reason", Required: true, Usage: "why the review was rejected"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := RejectReview(cmd.String("tasks-dir"), id, cmd.String("reason"))
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
