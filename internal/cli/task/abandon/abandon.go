// Package abandon owns the "abandon" command: its domain logic and CLI
// wiring.
package abandon

import (
	"fmt"

	"github.com/mhmdkzr/loop/internal/task"
)

// Abandon marks a task failed for good.
func Abandon(tasksDir, id, reason string) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.State == task.StateCompleted {
			return task.NotInState("task", string(t.State), "not already completed")
		}
		t.State = task.StateFailed
		t.FailureReason = reason
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	return t, nil
}
