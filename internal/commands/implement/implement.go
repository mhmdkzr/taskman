// Package implement owns the "implement" command: its domain logic, its
// CLI wiring, and its dispatch prompt.
package implement

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is implement's input. Shared verbatim by the CLI (cmd.go builds
// it from the positional id argument) and MCP (mcp.go uses it as the
// tool's input type directly) frontends; the jsonschema tag describes it
// to MCP clients.
type Request struct {
	ID string `json:"id" jsonschema:"the task id whose implementation attempt is done"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Implement marks a task's implementation attempt as done - the diff lives
// in the worktree, not the task file.
func Implement(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("implement task: %w", err)
	}
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if t.Status.Specification.State != task.StageDone {
			return task.NotInState("specification", string(t.Status.Specification.State), "done")
		}
		if t.Status.Implementation.State == task.StageDone {
			return task.NotInState("implementation", string(t.Status.Implementation.State), "!= done")
		}
		now := task.Now()
		t.Status.Implementation = task.StageStatus{State: task.StageDone, CompletedAt: &now}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("implement task: %w", err)
	}
	return t, nil
}
