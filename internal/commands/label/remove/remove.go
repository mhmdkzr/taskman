// Package remove owns the "label remove" command: it deletes one or more of
// a task's labels.
package remove

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is remove's input.
type Request struct {
	ID   uuid.UUID `json:"id"   jsonschema:"the task whose labels are being removed"`
	Keys []string  `json:"keys" jsonschema:"label keys to delete"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if len(r.Keys) == 0 {
		return fmt.Errorf("at least one key is required")
	}
	return nil
}

// Remove deletes req's keys from req's task's labels and returns the
// resulting task.
func Remove(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("remove labels: %w", err)
	}
	event := task.LabelsUpdated{Remove: req.Keys, At: time.Now().UTC()}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("remove labels: %w", err)
	}
	return t, nil
}
