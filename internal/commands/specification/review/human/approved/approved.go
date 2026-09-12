// Package approved owns the "specification review human approved" command.
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
	ID      uuid.UUID `json:"id"                jsonschema:"the task whose specification's human review was approved"`
	Comment string    `json:"comment,omitempty" jsonschema:"an optional approval comment"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Approved reports req's specification human review as approved and
// returns the resulting task.
func Approved(st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("approve specification human review: %w", err)
	}
	event := task.SpecificationReviewHumanApproved{Comment: req.Comment, At: time.Now().UTC()}
	t, err := st.Append(req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("approve specification human review: %w", err)
	}
	return t, nil
}
