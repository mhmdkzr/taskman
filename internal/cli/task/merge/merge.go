// Package merge owns the "merge" command: its domain logic and CLI wiring.
package merge

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is task merge's input.
type Request struct {
	Commit string
}

// Merge records that the caller already merged the task's branch.
func Merge(tasksDir, id string, req Request) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.Status.Review.State != task.StageDone {
			return task.NotInState("review", string(t.Status.Review.State), "done")
		}
		t.Status.Merge = task.StageStatus{State: task.StageDone, CompletedAt: new(task.Now())}
		t.State = task.StateCompleted
		if req.Commit != "" && t.Git.Commit != nil {
			t.Git.Commit.Hash = req.Commit
		}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("merge task: %w", err)
	}
	return t, nil
}
