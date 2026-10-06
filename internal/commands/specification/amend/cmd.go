package amend

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/task/view"
	"github.com/mhmdkzr/taskman/internal/utils"
)

// Command returns the "specification amend" command. Its flags are the same
// set specified declares; a flag that is not set is left out of the patch, so
// the task keeps its current value for that field.
func Command() *cli.Command {
	specFlags := utils.ReviewFlags("")
	implFlags := utils.ReviewFlags("impl-")
	flags := make([]cli.Flag, 0, 12+len(specFlags)+len(implFlags))
	flags = append(flags,
		&cli.StringFlag{Name: "id", Usage: "the task whose specification is being amended"},
		&cli.StringFlag{Name: "plan", Usage: "replacement plan; omit to keep the current plan"},
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
		Name:  "amend",
		Usage: "revise a task's specification before its implementation is recorded",
		Flags: flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			id, err := utils.IDFrom(cmd)
			if err != nil {
				return utils.Fail(err)
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

			t, err := Amend(ctx, st, requestFrom(cmd, id))
			if err != nil {
				return utils.Fail(err)
			}
			return view.PrintTask(cmd, t)
		},
	}
}

// requestFrom builds the patch from the flags the caller actually set, so an
// omitted flag keeps the task's current value.
func requestFrom(cmd *cli.Command, id uuid.UUID) Request {
	return Request{
		ID:                   id,
		Plan:                 stringPtr(cmd, "plan"),
		Review:               reviewPatchFrom(cmd, ""),
		ImplementationReview: reviewPatchFrom(cmd, "impl-"),
		Verification:         verificationPatchFrom(cmd),
		Worktree:             worktreePatchFrom(cmd),
	}
}

func reviewPatchFrom(cmd *cli.Command, prefix string) *ReviewPatch {
	patch := ReviewPatch{}
	if agent := (AgentReviewPatch{
		Required:    boolPtr(cmd, prefix+"agent-review"),
		UseSubagent: boolPtr(cmd, prefix+"agent-review-use-subagent"),
		AutoFix:     autoFixPatchFrom(cmd, prefix+"agent-review-auto-fix"),
	}); !agent.empty() {
		patch.Agent = &agent
	}
	if human := (HumanReviewPatch{
		Required: boolPtr(cmd, prefix+"human-review"),
		AutoFix:  autoFixPatchFrom(cmd, prefix+"human-review-auto-fix"),
	}); !human.empty() {
		patch.Human = &human
	}
	if patch.empty() {
		return nil
	}
	return &patch
}

func autoFixPatchFrom(cmd *cli.Command, prefix string) *AutoFixPatch {
	patch := AutoFixPatch{
		Enabled:     boolPtr(cmd, prefix),
		MaxRounds:   intPtr(cmd, prefix+"-max-rounds"),
		UseSubagent: boolPtr(cmd, prefix+"-use-subagent"),
	}
	if patch.empty() {
		return nil
	}
	return &patch
}

func verificationPatchFrom(cmd *cli.Command) *VerificationPatch {
	patch := VerificationPatch{
		Linters: boolPtr(cmd, "linters"),
		AutoFix: autoFixPatchFrom(cmd, "verification-auto-fix"),
	}
	if tests := (TestPatch{
		Unit:        boolPtr(cmd, "unit"),
		Integration: boolPtr(cmd, "integration"),
		EndToEnd:    boolPtr(cmd, "end-to-end"),
	}); !tests.empty() {
		patch.Tests = &tests
	}
	if patch.empty() {
		return nil
	}
	return &patch
}

func worktreePatchFrom(cmd *cli.Command) *WorktreePatch {
	patch := WorktreePatch{
		UseWorktree: boolPtr(cmd, "use-worktree"),
		Worktree:    stringPtr(cmd, "worktree"),
		Branch:      stringPtr(cmd, "branch"),
	}
	if patch.empty() {
		return nil
	}
	return &patch
}

func boolPtr(cmd *cli.Command, name string) *bool {
	if !cmd.IsSet(name) {
		return nil
	}
	value := cmd.Bool(name)
	return &value
}

func intPtr(cmd *cli.Command, name string) *int {
	if !cmd.IsSet(name) {
		return nil
	}
	value := cmd.Int(name)
	return &value
}

func stringPtr(cmd *cli.Command, name string) *string {
	if !cmd.IsSet(name) {
		return nil
	}
	value := cmd.String(name)
	return &value
}
