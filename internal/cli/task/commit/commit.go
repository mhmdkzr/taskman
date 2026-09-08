// Package commit owns the "commit" command: its domain logic and CLI wiring.
package commit

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is task commit's input - just which commit to read, if not HEAD.
type Request struct {
	Commit string
}

// Commit reads the caller's already-made commit directly out of the
// worktree via git log, rather than trusting reported text - design.md §6
// "When the commit happens". Called once right after verification first
// passes, and again each time a review-reject-recovery cycle clears.
func Commit(ctx context.Context, tasksDir string, git *task.GitClient, id string, req Request) (task.Task, error) {
	t, err := task.ReadTask(tasksDir, id)
	if err != nil {
		return task.Task{}, fmt.Errorf("read task: %w", err)
	}
	if t.Status.Verification.State != task.StageDone {
		return task.Task{}, fmt.Errorf(
			"verification precondition: %w",
			task.NotInState("verification", string(t.Status.Verification.State), "done"),
		)
	}
	commit, err := git.ReadCommit(ctx, t.Git.Worktree, req.Commit)
	if err != nil {
		return task.Task{}, fmt.Errorf("read commit: %w", err)
	}
	commit.At = task.Now()
	updated, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.Status.Verification.State != task.StageDone {
			return task.NotInState("verification", string(t.Status.Verification.State), "done")
		}
		t.Git.Commit = &commit
		t.Status.Review.State = task.StagePending
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("commit task: %w", err)
	}
	return updated, nil
}
