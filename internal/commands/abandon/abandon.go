// Package abandon owns the "abandon" command: its domain logic and CLI
// wiring.
package abandon

import (
	"context"
	"fmt"

	"github.com/mhmdkzr/taskman/internal/git"
	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
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

// Abandon marks a task abandoned for good, then commits the task's own now-
// terminal file itself through the Git shell.
func Abandon(ctx context.Context, tasksDir string, gitClient *git.Client, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, task.Abandoned{Reason: req.Reason})
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	if err := git.RecordBookkeeping(ctx, gitClient, tasksDir, t, "abandonment"); err != nil {
		return task.Task{}, fmt.Errorf("abandon task: %w", err)
	}
	return t, nil
}
