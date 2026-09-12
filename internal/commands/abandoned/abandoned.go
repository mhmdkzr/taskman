// Package abandoned owns the "abandoned" command: it ends a task
// unsuccessfully.
package abandoned

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is abandoned's input.
type Request struct {
	ID     uuid.UUID `json:"id"     jsonschema:"the task being abandoned"`
	Reason string    `json:"reason" jsonschema:"why the task is being abandoned"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// Abandoned ends req's task unsuccessfully and returns the resulting task.
func Abandoned(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	event := task.Abandoned{Reason: req.Reason, At: time.Now().UTC()}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	return t, nil
}
