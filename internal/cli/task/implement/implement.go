// Package implement owns the "implement" command: its domain logic, its
// CLI wiring, and its dispatch prompt.
package implement

import (
	"fmt"

	"github.com/mhmdkzr/loop/internal/task"
)

// Implement marks a task's implementation attempt as done - the diff lives
// in the worktree, not the task file.
func Implement(tasksDir, id string) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.Status.Specification.State != task.StageDone {
			return task.NotInState("specification", string(t.Status.Specification.State), "done")
		}
		if t.Status.Implementation.State == task.StageDone {
			return task.NotInState("implementation", string(t.Status.Implementation.State), "!= done")
		}
		now := task.Now()
		t.Status.Implementation = task.StageStatus{State: task.StageDone, CompletedAt: &now}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("implement task: %w", err)
	}
	return t, nil
}
