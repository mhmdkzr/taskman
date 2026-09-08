// Package abandon owns the "abandon" command: its domain logic and CLI
// wiring.
package abandon

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is abandon's input. Shared verbatim by the CLI (cmd.go builds it
// from flags) and MCP (mcp.go uses it as the tool's input type directly)
// frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID     string `json:"id"     jsonschema:"the task id being abandoned"`
	Reason string `json:"reason" jsonschema:"why the task is being abandoned"`
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

// Abandon marks a task failed for good.
func Abandon(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if t.State == task.StateCompleted {
			return task.NotInState("task", string(t.State), "not already completed")
		}
		t.State = task.StateFailed
		t.FailureReason = req.Reason
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	return t, nil
}
