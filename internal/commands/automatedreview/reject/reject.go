// Package reject owns the "automated-review rejected" command.
package reject

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is automated-review rejected's shared CLI and MCP input.
type Request struct {
	ID       string         `json:"id"       jsonschema:"the task whose automated review is rejected"`
	Findings []task.Finding `json:"findings" jsonschema:"findings against the review, each with a file and full detail text"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if len(r.Findings) == 0 {
		return fmt.Errorf("at least one finding is required")
	}
	return nil
}

// Reject records automated review findings.
func Reject(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("reject automated review: %w", err)
	}
	event := task.AutomatedReviewRecorded{Findings: req.Findings, At: time.Now().UTC()}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("reject automated review: %w", err)
	}
	return t, nil
}
