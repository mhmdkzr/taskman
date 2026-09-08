package task

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/cli/support"
	"github.com/mhmdkzr/loop/internal/task"
)

// Next returns the "next" command.
func Next() *cli.Command {
	return &cli.Command{
		Name:      "next",
		Usage:     "show what should happen next for this task",
		ArgsUsage: "<id>",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			g, err := task.Next(support.RepoFrom(cmd), id)
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintGuidance(cmd, g)
		},
	}
}
