// Package get owns the "get" command: its domain logic and CLI wiring.
package get

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is get's input. Shared verbatim by the CLI (cmd.go builds it from
// the positional id argument) and MCP (mcp.go uses it as the tool's input
// type directly) frontends; the jsonschema tag describes it to MCP clients.
type Request struct {
	ID string `json:"id" jsonschema:"the task id to read"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Get reads the current state of req.ID under tasksDir.
func Get(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("get task: %w", err)
	}
	t, err := store.Read(tasksDir, req.ID)
	if err != nil {
		return task.Task{}, fmt.Errorf("read task: %w", err)
	}
	return t, nil
}
