package specificationreject

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "rejected" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "rejected",
		Usage:     "record a human's rejection of a drafted specification",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "reason", Required: true, Usage: "why the specification was rejected"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Reject(cmd.String("tasks-dir"), Request{ID: id, Reason: cmd.String("reason")})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
