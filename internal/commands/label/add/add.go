// Package add owns the "label add" command: it sets or overwrites one or
// more of a task's labels.
package add

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is add's input.
type Request struct {
	ID     uuid.UUID         `json:"id"     jsonschema:"the task whose labels are being set"`
	Labels map[string]string `json:"labels" jsonschema:"labels to set as key/value pairs - overwrites any existing value for the same key"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if len(r.Labels) == 0 {
		return fmt.Errorf("at least one label is required")
	}
	return nil
}

// Add sets req's labels on req's task and returns the resulting task.
func Add(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("add labels: %w", err)
	}
	event := task.LabelsUpdated{Set: req.Labels, At: time.Now().UTC()}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("add labels: %w", err)
	}
	return t, nil
}
