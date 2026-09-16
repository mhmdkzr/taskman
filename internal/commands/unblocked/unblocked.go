// Package unblocked owns the "unblocked" command: it resumes a blocked task,
// optionally granting more auto-fix rounds to the gate that exhausted its
// budget.
package unblocked

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is unblocked's input.
type Request struct {
	ID     uuid.UUID `json:"id"               jsonschema:"the task being unblocked"`
	Reason string    `json:"reason"           jsonschema:"why the task is being unblocked"`
	Rounds int       `json:"rounds,omitempty" jsonschema:"additional auto-fix rounds granted; required for a budget-exhaustion blockage, must be omitted for an escalation"`
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

// Unblocked resumes req's task and returns the resulting task.
func Unblocked(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("unblock task: %w", err)
	}
	event := task.Unblocked{Reason: req.Reason, Rounds: req.Rounds, At: time.Now().UTC()}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("unblock task: %w", err)
	}
	return t, nil
}
