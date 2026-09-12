// Package escalated owns the "escalated" command: it blocks a task pending
// outside intervention.
package escalated

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is escalated's input.
type Request struct {
	ID     uuid.UUID `json:"id"     jsonschema:"the task being escalated"`
	Stage  string    `json:"stage"  jsonschema:"the stage the task is stuck at"`
	Reason string    `json:"reason" jsonschema:"why the task is stuck"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if r.Stage == "" {
		return fmt.Errorf("stage is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// Escalated blocks req's task and returns the resulting task.
func Escalated(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("escalate task: %w", err)
	}
	event := task.Escalated{Stage: req.Stage, Reason: req.Reason, At: time.Now().UTC()}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("escalate task: %w", err)
	}
	return t, nil
}
