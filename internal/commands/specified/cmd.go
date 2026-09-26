package specified

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "specified" command.
func Command() *cli.Command {
	specFlags := utils.ReviewFlags("")
	implFlags := utils.ReviewFlags("impl-")
	flags := make([]cli.Flag, 0, 12+len(specFlags)+len(implFlags))
	flags = append(flags,
		&cli.StringFlag{Name: "id", Usage: "the task whose specification was submitted"},
		&cli.StringFlag{Name: "plan", Usage: "the specification's plan"},
		&cli.BoolFlag{Name: "unit", Usage: "require unit tests"},
		&cli.BoolFlag{Name: "integration", Usage: "require integration tests"},
		&cli.BoolFlag{Name: "end-to-end", Usage: "require end-to-end tests"},
		&cli.BoolFlag{Name: "linters", Usage: "require linters"},
		&cli.BoolFlag{Name: "verification-auto-fix", Usage: "automatically fix verification failures"},
		&cli.IntFlag{Name: "verification-auto-fix-max-rounds", Usage: "max verification auto-fix rounds"},
		&cli.BoolFlag{Name: "verification-auto-fix-use-subagent", Usage: "run verification auto-fix in a subagent"},
		&cli.BoolFlag{Name: "use-worktree", Usage: "implement this task in a fresh worktree"},
		&cli.StringFlag{Name: "worktree", Usage: "the worktree path to use, if --use-worktree"},
		&cli.StringFlag{Name: "branch", Usage: "the branch name to use, if --use-worktree"},
	)
	flags = append(flags, specFlags...)
	flags = append(flags, implFlags...)

	return &cli.Command{
		Name:  "specified",
		Usage: "report a task's specification as submitted",
		Flags: flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
			}
			if err := utils.RequireFlags(cmd, "plan"); err != nil {
				return err
			}
			if err := utils.RequireFlagsIf(cmd, cmd.Bool("use-worktree"), "worktree", "branch"); err != nil {
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

			t, err := Specified(ctx, st, Request{
				ID:     id,
				Plan:   cmd.String("plan"),
				Review: utils.ReviewConfigurationFrom(cmd, ""),
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
				ImplementationReview: utils.ReviewConfigurationFrom(cmd, "impl-"),
				Worktree: task.WorktreePolicy{
					UseWorktree: cmd.Bool("use-worktree"),
					Worktree:    cmd.String("worktree"),
					Branch:      cmd.String("branch"),
				},
			})
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}
