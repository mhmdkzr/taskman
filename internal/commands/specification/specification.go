// Package specification owns the "specified" command: its domain logic, CLI
// wiring, and dispatch prompt.
package specification

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is specification's input. Shared verbatim by the CLI (cmd.go builds it
// from flags) and MCP (mcp.go uses it as the tool's input type directly)
// frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID       string `json:"id"        jsonschema:"the task id to specify"`
	Result   string `json:"result"    jsonschema:"the drafted specification"`
	DoneWhen string `json:"done_when" jsonschema:"the drafted acceptance criteria"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if r.Result == "" {
		return fmt.Errorf("result is required")
	}
	if r.DoneWhen == "" {
		return fmt.Errorf("done_when is required")
	}
	return nil
}

// Specify writes a task's specification and done_when, drafted from its
// definition.
func Specify(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("submit specification: %w", err)
	}
	event := task.SpecificationSubmitted{Specification: req.Result, DoneWhen: req.DoneWhen}
	t, err := store.Update(tasksDir, req.ID, func(current task.Task) (task.Task, error) {
		return task.Apply(current, event)
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("submit specification: %w", err)
	}
	return t, nil
}
