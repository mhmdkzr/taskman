// Package approve owns the "specification approved" command.
package approve

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is specification-approved's shared CLI and MCP input.
type Request struct {
	ID      string `json:"id"                jsonschema:"the task whose specification is being approved"`
	Comment string `json:"comment,omitempty" jsonschema:"an optional approval note"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Approve records a human's approval of a drafted specification.
func Approve(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("approve specification: %w", err)
	}
	event := task.SpecificationApproved{Comment: req.Comment, At: time.Now().UTC()}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("approve specification: %w", err)
	}
	return t, nil
}
