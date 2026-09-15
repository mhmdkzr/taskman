// Package prune owns the "prune" command: it permanently removes every
// completed task and its event log.
package prune

import (
	"context"
	"fmt"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is prune's input.
type Request struct {
	DryRun bool `json:"dry-run" jsonschema:"report which tasks would be pruned without deleting them"`
}

// PrunedTask is one completed task selected for pruning.
type PrunedTask struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

// Result reports which completed tasks were pruned (or, under DryRun, which
// would have been).
type Result struct {
	Pruned []PrunedTask `json:"pruned"`
	DryRun bool         `json:"dry-run"`
}

// Prune permanently removes every task in the completed state, along with
// its event log, and returns their ids and titles. Under req.DryRun it
// reports the same tasks but deletes nothing. A task whose events can no
// longer be replayed is never pruned - its state cannot be established - and
// the replay error is returned.
func Prune(ctx context.Context, st *store.Store, req Request) (Result, error) {
	ids, err := st.List(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("prune tasks: %w", err)
	}

	completed := make([]PrunedTask, 0, len(ids))
	for _, id := range ids {
		current, err := st.Read(ctx, id)
		if err != nil {
			return Result{}, fmt.Errorf("prune tasks: %w", err)
		}
		if current.State() == task.StateCompleted {
			completed = append(completed, PrunedTask{ID: id, Title: current.Definition.Title})
		}
	}
	if !req.DryRun {
		for _, pruned := range completed {
			if err := st.Delete(ctx, pruned.ID); err != nil {
				return Result{}, fmt.Errorf("prune tasks: %w", err)
			}
		}
	}
	return Result{Pruned: completed, DryRun: req.DryRun}, nil
}
