package reject

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
		Usage:     "record automated review findings",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "finding", Required: true, Usage: "a finding as file=detail - repeatable"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			findings, err := utils.ParseFindings(cmd.StringSlice("finding"))
			if err != nil {
				return cli.Exit(err, 2)
			}
			t, err := Reject(cmd.String("tasks-dir"), Request{ID: id, Findings: findings})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
