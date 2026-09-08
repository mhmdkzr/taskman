package specify

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/cli/support"
)

// Command returns the "specify" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:      "specify",
		Usage:     "write a task's specification and acceptance criteria",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "result", Required: true, Usage: "the drafted specification"},
			&cli.StringFlag{Name: "done-when", Required: true, Usage: "the drafted acceptance criteria"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := support.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Specify(cmd.String("tasks-dir"), id, Request{
				Result:   cmd.String("result"),
				DoneWhen: cmd.String("done-when"),
			})
			if err != nil {
				return support.Fail(err)
			}
			return support.PrintTask(cmd, t)
		},
	}
}
