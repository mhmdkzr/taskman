// Package committed owns the "committed" command: it records the task's
// implementation worktree's current commit.
package committed

import (
	"context"
	"fmt"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is committed's input.
type Request struct {
	ID uuid.UUID `json:"id" jsonschema:"the task whose commit was recorded"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Committed reads req's implementation worktree's current commit and
// records it, returning the resulting task.
func Committed(ctx context.Context, st *store.Store, gitClient *git.Client, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record commit: %w", err)
	}
	current, err := st.Read(req.ID)
	if err != nil {
		return task.Task{}, fmt.Errorf("record commit: %w", err)
	}
	if current.Implementation == nil {
		return task.Task{}, fmt.Errorf("record commit: implementation not yet reported")
	}
	commit, err := gitClient.ReadCommit(ctx, current.Implementation.Git.Worktree, "")
	if err != nil {
		return task.Task{}, fmt.Errorf("record commit: %w", err)
	}
	t, err := st.Append(req.ID, task.CommitRecorded{Commit: commit, At: time.Now().UTC()})
	if err != nil {
		return task.Task{}, fmt.Errorf("record commit: %w", err)
	}
	return t, nil
}
