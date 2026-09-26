// Package specified owns the "specified" command: it reports a task's
// specification as submitted.
package specified

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is specified's input.
//
//nolint:modernize // omitempty marks these fields optional in the reflected MCP schema; omitzero would not.
type Request struct {
	ID     uuid.UUID                `json:"id"               jsonschema:"the task whose specification was submitted"`
	Plan   string                   `json:"plan"             jsonschema:"the specification's plan"`
	Review task.ReviewConfiguration `json:"review,omitempty" jsonschema:"which specification-review gates this specification requires"`

	// Verification, ImplementationReview, and Worktree are the implementation-
	// time requirement/policy this specification fixes up front, so a later
	// `implemented` call is pure fact-reporting: it copies these forward
	// instead of accepting them as new input.
	Verification         task.Verification        `json:"verification,omitempty"          jsonschema:"which verification checks the implementation will require"`
	ImplementationReview task.ReviewConfiguration `json:"implementation-review,omitempty" jsonschema:"which review gates the implementation will require"`
	Worktree             task.WorktreePolicy      `json:"worktree,omitempty"              jsonschema:"whether the implementation uses a fresh worktree, and where"`
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
func Specified(ctx context.Context, st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record specification: %w", err)
	}
	event := task.SpecificationSubmitted{
		Specification: task.Specification{
			Plan:                 req.Plan,
			Review:               req.Review,
			Verification:         req.Verification,
			ImplementationReview: req.ImplementationReview,
			Worktree:             req.Worktree,
		},
		At: time.Now().UTC(),
	}
	t, err := st.Append(ctx, req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("record specification: %w", err)
	}
	return t, nil
}
