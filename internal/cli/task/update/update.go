// Package update owns the "update" command: its domain logic and CLI wiring.
package update

import (
	"fmt"
	"maps"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is task update's input - patch semantics, only non-nil fields are
// applied. design.md §6.
type Request struct {
	Title           *string
	SetLabels       map[string]string
	UnsetLabels     []string
	References      []string
	ClearReferences bool
}

// Update patches a task's metadata (title, labels, references). It never
// touches specification/done_when (own command: specify) or git/status
// (taskman-managed). No precondition on State/Status - metadata isn't
// workflow state. design.md §6.
func Update(tasksDir, id string, req Request) (task.Task, error) {
	merged := make(map[string]string)
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if req.Title != nil {
			t.Title = *req.Title
		}
		if len(req.SetLabels) > 0 || len(req.UnsetLabels) > 0 {
			maps.Copy(merged, t.Labels)
			maps.Copy(merged, req.SetLabels)
			for _, k := range req.UnsetLabels {
				delete(merged, k)
			}
			if err := task.ValidateLabels(merged); err != nil {
				return fmt.Errorf("validate labels: %w", err)
			}
			t.Labels = merged
		}
		if req.ClearReferences {
			t.References = nil
		} else if len(req.References) > 0 {
			t.References = req.References
		}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("update task: %w", err)
	}
	return t, nil
}
