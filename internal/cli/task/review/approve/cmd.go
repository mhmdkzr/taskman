package approve

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
)

// Command returns the "approve" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "approve",
		Usage:     "record a human's approval at the review stage",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "comment", Usage: "an optional note, e.g. LGTM"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := ApproveReview(cmd.String("tasks-dir"), id, cmd.String("comment"))
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
