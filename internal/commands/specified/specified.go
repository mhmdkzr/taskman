// Package specified owns the "specified" command: it reports a task's
// specification as submitted.
package specified

import (
	"fmt"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is specified's input.
type Request struct {
	ID     uuid.UUID                `json:"id"   jsonschema:"the task whose specification was submitted"`
	Plan   string                   `json:"plan" jsonschema:"the specification's plan"`
	Review task.ReviewConfiguration `json:"review,omitempty" jsonschema:"which review gates this specification requires"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if r.Plan == "" {
		return fmt.Errorf("plan is required")
	}
	return nil
}

// Specified reports req's specification as submitted and returns the
// resulting task.
func Specified(st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record specification: %w", err)
	}
	event := task.SpecificationSubmitted{
		Specification: task.Specification{Plan: req.Plan, Review: req.Review},
		At:            time.Now().UTC(),
	}
	t, err := st.Append(req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("record specification: %w", err)
	}
	return t, nil
}
