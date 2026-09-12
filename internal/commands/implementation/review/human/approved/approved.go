// Package approved owns the "implementation review human approved"
// command.
package approved

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is approved's input.
type Request struct {
	ID      uuid.UUID `json:"id"                jsonschema:"the task whose implementation's human review was approved"`
	Comment string    `json:"comment,omitempty" jsonschema:"an optional approval comment"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Approved reports req's implementation human review as approved and
// returns the resulting task.
func Approved(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("approve implementation human review: %w", err)
	}
	event := task.ImplementationReviewHumanApproved{Comment: req.Comment, At: time.Now().UTC()}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("approve implementation human review: %w", err)
	}
	return t, nil
}
