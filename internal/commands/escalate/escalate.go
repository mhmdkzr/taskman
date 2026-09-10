// Package escalate owns the "escalate" command: its domain logic and CLI
// wiring.
package escalate

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
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
	event := task.Escalated{Stage: req.Stage, Reason: req.Reason, At: time.Now().UTC()}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("escalate task: %w", err)
	}
	return t, nil
}
