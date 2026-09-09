// Package escalate owns the "escalate" command: its domain logic and CLI
// wiring.
package escalate

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is escalate's input. Shared verbatim by the CLI (cmd.go builds it
// from flags) and MCP (mcp.go uses it as the tool's input type directly)
// frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID     string `json:"id"     jsonschema:"the task id being escalated"`
	Stage  string `json:"stage"  jsonschema:"which stage was in flight (definition, specification, implementation, verification, review, or merge)"`
	Reason string `json:"reason" jsonschema:"why the agent gave up"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if r.Stage == "" {
		return fmt.Errorf("stage is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// Escalate blocks a task on a caller's own report that a dispatched agent
// gave up rather than keep iterating.
func Escalate(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("escalate task: %w", err)
	}
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if t.State == task.StateCompleted || t.State == task.StateFailed {
			return task.NotInState("task", string(t.State), "not already terminal")
		}
		t.State = task.StateBlocked
		t.Blocked = &task.Blocked{Stage: req.Stage, Reason: req.Reason, At: task.Now()}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("escalate task: %w", err)
	}
	return t, nil
}
