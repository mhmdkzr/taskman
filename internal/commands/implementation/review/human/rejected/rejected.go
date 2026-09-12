// Package rejected owns the "implementation review human rejected"
// command.
package rejected

import (
	"context"
	"fmt"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is rejected's input.
type Request struct {
	ID     uuid.UUID `json:"id"     jsonschema:"the task whose implementation's human review was rejected"`
	Reason string    `json:"reason" jsonschema:"why the implementation's human review was rejected"`
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

// Rejected reports req's implementation human review as rejected and
// returns the resulting task.
func Rejected(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("reject implementation human review: %w", err)
	}
	event := task.ImplementationReviewHumanRejected{Reason: req.Reason, At: time.Now().UTC()}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("reject implementation human review: %w", err)
	}
	return t, nil
}
