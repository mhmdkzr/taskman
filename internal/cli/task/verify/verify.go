// Package verify owns the "verify" command: its domain logic and CLI wiring.
package verify

import (
	"fmt"

	"github.com/mhmdkzr/loop/internal/task"
)

// Request is task verify's input - one reported build-check attempt.
type Request struct {
	Checks map[string]task.CheckResult
	Output string
}

// Verify appends one build-check attempt to a task's verifications log -
// design.md §6. It never sets verification.state itself; that only happens
// via task review record's approval.
func Verify(tasksDir, id string, req Request) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.Status.Implementation.State != task.StageDone {
			return task.NotInState("implementation", string(t.Status.Implementation.State), "done")
		}
		if err := task.NotBlockedOrFailed(t); err != nil {
			return fmt.Errorf("blocked/failed precondition: %w", err)
		}
		t.Verifications = append(t.Verifications, task.Verification{
			Checks:    req.Checks,
			Output:    req.Output,
			CreatedAt: task.Now(),
		})
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("verify task: %w", err)
	}
	return t, nil
}
