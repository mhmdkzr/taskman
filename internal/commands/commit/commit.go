// Package commit owns the "commit" command: its domain logic and CLI wiring.
package commit

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is commit's input - which task to record a commit for, and which
// commit to read, if not HEAD. Shared verbatim by the CLI (cmd.go builds it
// from flags) and MCP (mcp.go uses it as the tool's input type directly)
// frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID     string `json:"id"               jsonschema:"the task id the commit belongs to"`
	Commit string `json:"commit,omitempty" jsonschema:"read this commit-ish instead of HEAD"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Commit reads the caller's already-made commit directly out of the
// worktree via git log, rather than trusting reported text - design.md §6
// "When the commit happens". Called once right after verification first
// passes, and again each time a review-reject-recovery cycle clears.
func Commit(ctx context.Context, tasksDir string, git *task.GitClient, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("commit task: %w", err)
	}
	t, err := task.ReadTask(tasksDir, req.ID)
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
	updated, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
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
