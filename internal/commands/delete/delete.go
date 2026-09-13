// Package delete owns the "delete" command: it permanently removes a task
// and its event log.
package delete

import (
	"context"
	"fmt"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is delete's input.
type Request struct {
	ID uuid.UUID `json:"id" jsonschema:"the task id to delete"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Result confirms which task was deleted.
type Result struct {
	ID uuid.UUID `json:"id"`
}

// Delete permanently removes req's task and its entire event log. Unlike
// every other command it records no event and never replays the task's log,
// so any task - including one whose events can no longer be replayed - can be
// deleted. It fails with store.ErrTaskNotFound if no task has that id.
func Delete(ctx context.Context, st *store.Store, req Request) (Result, error) {
	if err := req.validate(); err != nil {
		return Result{}, fmt.Errorf("delete task: %w", err)
	}
	if err := st.Delete(ctx, req.ID); err != nil {
		return Result{}, fmt.Errorf("delete task: %w", err)
	}
	return Result(req), nil
}
