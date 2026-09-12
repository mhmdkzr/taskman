// Package approved owns the "implementation review agent approved"
// command.
package approved

import (
	"fmt"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is approved's input.
type Request struct {
	ID      uuid.UUID `json:"id"                jsonschema:"the task whose implementation's automated review was approved"`
	Comment string    `json:"comment,omitempty" jsonschema:"an optional approval comment"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Approved reports req's implementation automated review as approved and
// returns the resulting task.
func Approved(st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("approve implementation automated review: %w", err)
	}
	event := task.ImplementationReviewAgentApproved{Comment: req.Comment, At: time.Now().UTC()}
	t, err := st.Append(req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("approve implementation automated review: %w", err)
	}
	return t, nil
}
