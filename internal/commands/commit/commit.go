// Package commit owns the "committed" command: its domain logic and CLI wiring.
package commit

import (
	"context"
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
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
// worktree via git log, rather than trusting reported text. Called once
// right after verification first passes, and again each time a
// review-reject-recovery cycle clears. For an auto-approve trunk task, this
// call completes the task outright, so it's also where its own now-terminal
// file gets committed by the Git shell.
func Commit(ctx context.Context, tasksDir string, gitClient *git.Client, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("commit task: %w", err)
	}
	t, err := store.Read(tasksDir, req.ID)
	if err != nil {
		return task.Task{}, fmt.Errorf("read task: %w", err)
	}
	if err := task.Accepts(t, task.EventCommitRecorded); err != nil {
		return task.Task{}, fmt.Errorf("commit precondition: %w", err)
	}
	commit, err := gitClient.ReadCommit(ctx, t.Git.Worktree, req.Commit)
	if err != nil {
		return task.Task{}, fmt.Errorf("read commit: %w", err)
	}
	commit.At = time.Now().UTC()
	updated, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, task.CommitRecorded{Commit: commit})
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("commit task: %w", err)
	}
	if updated.State == task.StateCompleted {
		if err := git.RecordBookkeeping(ctx, gitClient, tasksDir, updated, "completion"); err != nil {
			return task.Task{}, fmt.Errorf("commit task: %w", err)
		}
	}
	return updated, nil
}
