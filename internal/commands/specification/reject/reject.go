// Package reject owns the "specification rejected" command.
package reject

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is specification-rejected's shared CLI and MCP input.
type Request struct {
	ID     string `json:"id"     jsonschema:"the task whose specification is being rejected"`
	Reason string `json:"reason" jsonschema:"why the specification was rejected"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// Reject records a human's rejection of a drafted specification.
func Reject(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("reject specification: %w", err)
	}
	event := task.SpecificationRejected{Reason: req.Reason, At: time.Now().UTC()}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("reject specification: %w", err)
	}
	return t, nil
}
