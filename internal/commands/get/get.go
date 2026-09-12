// Package get owns the "get" command: it reads one task's current state.
package get

import (
	"context"
	"fmt"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is get's input.
type Request struct {
	ID uuid.UUID `json:"id" jsonschema:"the task id to read"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Get reads req's task.
func Get(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("get task: %w", err)
	}
	t, err := st.Read(ctx, req.ID)
	if err != nil {
		return task.Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}
