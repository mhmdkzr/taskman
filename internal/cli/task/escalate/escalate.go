// Package escalate owns the "escalate" command: its domain logic and CLI
// wiring.
package escalate

import (
	"fmt"

	"github.com/mhmdkzr/loop/internal/task"
)

// Request is task escalate's input.
type Request struct {
	Stage  string
	Reason string
}

// Escalate blocks a task on a caller's own report that a dispatched agent
// gave up rather than keep iterating - design.md §6's "Giving up".
func Escalate(tasksDir, id string, req Request) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.State == task.StateCompleted || t.State == task.StateFailed {
			return task.NotInState("task", string(t.State), "not already terminal")
		}
		t.State = task.StateBlocked
		t.Blocked = &task.Blocked{Stage: req.Stage, Reason: req.Reason, At: task.Now()}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("escalate task: %w", err)
	}
	return t, nil
}
