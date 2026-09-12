// Package rejected owns the "implementation review agent rejected"
// command.
package rejected

import (
	"fmt"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is rejected's input.
type Request struct {
	ID       uuid.UUID      `json:"id"       jsonschema:"the task whose implementation's automated review was rejected"`
	Findings []task.Finding `json:"findings" jsonschema:"findings against the review, each with a location and detail"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if len(r.Findings) == 0 {
		return fmt.Errorf("at least one finding is required")
	}
	return nil
}

// Rejected reports req's implementation automated review as rejected and
// returns the resulting task.
func Rejected(st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("reject implementation automated review: %w", err)
	}
	event := task.ImplementationReviewAgentRejected{Findings: req.Findings, At: time.Now().UTC()}
	t, err := st.Append(req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("reject implementation automated review: %w", err)
	}
	return t, nil
}
