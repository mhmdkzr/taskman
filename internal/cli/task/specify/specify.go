// Package specify owns the "specify" command: its domain logic, CLI
// wiring, and dispatch prompt.
package specify

import (
	"fmt"

	"github.com/mhmdkzr/loop/internal/task"
)

// Request is task specify's input.
type Request struct {
	Result   string
	DoneWhen string
}

// Specify writes a task's specification and done_when, drafted from its
// definition - design.md §6.
func Specify(tasksDir, id string, req Request) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.Status.Specification.State == task.StageDone {
			return task.NotInState("specification", string(t.Status.Specification.State), "!= done")
		}
		t.Specification = req.Result
		t.DoneWhen = req.DoneWhen
		t.Status.Specification = task.StageStatus{State: task.StageDone, CompletedAt: new(task.Now())}
		t.State = task.StateStarted
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("specify task: %w", err)
	}
	return t, nil
}
