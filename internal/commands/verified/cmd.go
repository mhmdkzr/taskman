package verified

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "verified" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "verified",
		Usage: "report one verification attempt",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Required: true, Usage: "the task whose verification was reported"},
			&cli.StringFlag{Name: "unit", Usage: "the unit test check's result: ok or error"},
			&cli.StringFlag{Name: "integration", Usage: "the integration test check's result: ok or error"},
			&cli.StringFlag{Name: "end-to-end", Usage: "the end-to-end test check's result: ok or error"},
			&cli.StringFlag{Name: "linters", Usage: "the linters check's result: ok or error"},
			&cli.StringFlag{Name: "output", Usage: "verification output/log text"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return err
			}
			var checks task.Checks
			for _, c := range []struct {
				flag string
				dest *task.CheckResult
			}{
				{"unit", &checks.Unit},
				{"integration", &checks.Integration},
				{"end-to-end", &checks.EndToEnd},
				{"linters", &checks.Linters},
			} {
				result, err := utils.ParseCheckResult(c.flag, cmd.String(c.flag))
				if err != nil {
					return err
				}
				*c.dest = result
			}

			st, err := utils.StoreFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			defer st.Close()

			t, err := Verified(ctx, st, Request{ID: id, Checks: checks, Output: cmd.String("output")})
			if err != nil {
				return utils.Fail(err)
			}
			return utils.PrintTask(cmd, t)
		},
	}
}
