package implemented

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "implemented" command.
func Command() *cli.Command {
	flags := make([]cli.Flag, 0, 10+len(utils.ReviewFlags()))
	flags = append(flags,
		&cli.StringFlag{Name: "id", Usage: "the task that was implemented"},
		&cli.StringFlag{Name: "worktree", Usage: "the worktree the implementation was done in"},
		&cli.StringFlag{Name: "branch", Usage: "the branch the implementation was done on"},
		&cli.BoolFlag{Name: "unit", Usage: "require unit tests"},
		&cli.BoolFlag{Name: "integration", Usage: "require integration tests"},
		&cli.BoolFlag{Name: "end-to-end", Usage: "require end-to-end tests"},
		&cli.BoolFlag{Name: "linters", Usage: "require linters"},
		&cli.BoolFlag{Name: "verification-auto-fix", Usage: "automatically fix verification failures"},
		&cli.IntFlag{Name: "verification-auto-fix-max-rounds", Usage: "max verification auto-fix rounds"},
		&cli.BoolFlag{Name: "verification-auto-fix-use-subagent", Usage: "run verification auto-fix in a subagent"},
	)
	flags = append(flags, utils.ReviewFlags()...)

	return &cli.Command{
		Name:  "implemented",
		Usage: "report a task's implementation as complete",
		Flags: flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			if err := utils.RequireFlags(cmd, "worktree", "branch"); err != nil {
				return err
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

			t, err := Implemented(ctx, st, Request{
				ID:       id,
				Worktree: cmd.String("worktree"),
				Branch:   cmd.String("branch"),
				Verification: task.Verification{
					Tests: task.TestConfiguration{
						Unit:        cmd.Bool("unit"),
						Integration: cmd.Bool("integration"),
						EndToEnd:    cmd.Bool("end-to-end"),
					},
					Linters: cmd.Bool("linters"),
					AutoFix: task.AutoFix{
						Enabled:     cmd.Bool("verification-auto-fix"),
						MaxRounds:   cmd.Int("verification-auto-fix-max-rounds"),
						UseSubagent: cmd.Bool("verification-auto-fix-use-subagent"),
					},
				},
				Review: utils.ReviewConfigurationFrom(cmd),
			})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
