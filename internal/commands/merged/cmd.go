package merged

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "merged" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "merged",
		Usage: "record the task's merge into its target branch",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task that was merged"},
			&cli.StringFlag{Name: "target", Required: true, Usage: "the branch it was merged into"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer utils.CloseStore(st)

			t, err := Merged(ctx, st, utils.GitFrom(cmd), Request{ID: id, Target: cmd.String("target")})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
