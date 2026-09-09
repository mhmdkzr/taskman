// Package abandon owns the "abandon" command: its domain logic and CLI
// wiring.
package abandon

import (
	"context"
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

// Abandon marks a task failed for good, then commits the task's own now-
// terminal file itself - see task.RecordBookkeeping.
func Abandon(ctx context.Context, tasksDir string, git *task.GitClient, req Request) (task.Task, error) {
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
	if err := task.RecordBookkeeping(ctx, git, tasksDir, t, "abandonment"); err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	return t, nil
}
