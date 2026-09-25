package verified

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// parseCheckResult parses a --unit/--integration/--end-to-end/--linters
// flag's value. An empty value means the check wasn't reported.
func parseCheckResult(flag, value string) (task.CheckResult, error) {
	switch task.CheckResult(value) {
	case "":
		return "", nil
	case task.CheckOK, task.CheckError:
		return task.CheckResult(value), nil
	default:
		return "", cli.Exit(
			fmt.Sprintf("--%s: value must be %q or %q, got %q", flag, task.CheckOK, task.CheckError, value),
			2,
		)
	}
}

// Command returns the "verified" command.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "verified",
		Usage: "report one verification attempt",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "the task whose verification was reported"},
			&cli.StringFlag{Name: "unit", Usage: "the unit test check's result: ok or error"},
			&cli.StringFlag{Name: "integration", Usage: "the integration test check's result: ok or error"},
			&cli.StringFlag{Name: "end-to-end", Usage: "the end-to-end test check's result: ok or error"},
			&cli.StringFlag{Name: "linters", Usage: "the linters check's result: ok or error"},
			&cli.StringFlag{Name: "output", Usage: "verification output/log text"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
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
				result, err := parseCheckResult(c.flag, cmd.String(c.flag))
				if err != nil {
					return err
				}
				*c.dest = result
			}

			st, err := store.Open(ctx, cmd.String("db"))
			if err != nil {
				return utils.Fail(err)
			}
			defer func() {
				if err := st.Close(); err != nil {
					slog.Error("close store", "error", err)
				}
			}()

			t, err := Verified(ctx, st, Request{ID: id, Checks: checks, Output: cmd.String("output")})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
