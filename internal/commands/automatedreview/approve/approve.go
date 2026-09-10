// Package approve owns the "automated-review approved" command.
package approve

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is automated-review approved's shared CLI and MCP input.
type Request struct {
	ID string `json:"id" jsonschema:"the task whose automated review is approved"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Approve records an automated review approval.
func Approve(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("approve automated review: %w", err)
	}
	event := task.AutomatedReviewRecorded{Approved: true, At: time.Now().UTC()}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("approve automated review: %w", err)
	}
	return t, nil
}
