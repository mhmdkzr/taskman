// Package merged owns the "merged" command: it records the task's merge
// into its target branch.
package merged

import (
	"context"
	"fmt"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is merged's input.
type Request struct {
	ID     uuid.UUID `json:"id"     jsonschema:"the task that was merged"`
	Target string    `json:"target" jsonschema:"the branch it was merged into"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if r.Target == "" {
		return fmt.Errorf("target is required")
	}
	return nil
}

// Merged reads req's implementation worktree's resulting commit and
// records the merge, returning the resulting task.
func Merged(ctx context.Context, st *store.Store, gitClient *git.Client, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record merge: %w", err)
	}
	current, err := st.Read(ctx, req.ID)
	if err != nil {
		return task.Task{}, fmt.Errorf("record merge: %w", err)
	}
	if current.Implementation == nil {
		return task.Task{}, fmt.Errorf("record merge: implementation not yet reported")
	}
	commit, err := gitClient.ReadCommit(ctx, current.Implementation.Git.Worktree, "")
	if err != nil {
		return task.Task{}, fmt.Errorf("record merge: %w", err)
	}
	event := task.MergeCompleted{
		Merge: task.GitMerge{Target: req.Target, Commit: commit.Hash, At: commit.At},
		At:    time.Now().UTC(),
	}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("record merge: %w", err)
	}
	return t, nil
}
