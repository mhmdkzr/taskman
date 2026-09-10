package specify

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/commands/specify/approve"
	"github.com/mhmdkzr/taskman/internal/commands/specify/reject"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command builds the "specification" command tree for human decisions.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "specification",
		Usage: "the specification approval stage",
		Commands: []*cli.Command{
			approve.Command(),
			reject.Command(),
		},
	}
}

// SpecifiedCommand returns the "specified" command that records an agent's
// drafted specification and acceptance criteria.
func SpecifiedCommand() *cli.Command {
	return &cli.Command{
		Name:      "specified",
		Usage:     "write a task's specification and acceptance criteria",
		ArgsUsage: "<id>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "result", Required: true, Usage: "the drafted specification"},
			&cli.StringFlag{Name: "done-when", Required: true, Usage: "the drafted acceptance criteria"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.RequireID(cmd)
			if err != nil {
				return fmt.Errorf("require id: %w", err)
			}
			t, err := Specify(cmd.String("tasks-dir"), Request{
				ID:       id,
				Result:   cmd.String("result"),
				DoneWhen: cmd.String("done-when"),
			})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
