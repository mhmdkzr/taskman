// Package delete owns the "delete" command: its domain logic and CLI
// wiring.
package delete

import (
	"errors"
	"fmt"
	"os"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is delete's input. Shared verbatim by the CLI (cmd.go builds it
// from the positional id argument) and MCP (mcp.go uses it as the tool's
// input type directly) frontends; the jsonschema tag describes it to MCP
// clients.
type Request struct {
	ID string `json:"id" jsonschema:"the task id to remove"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Delete removes req.ID's file outright under tasksDir - no soft-delete,
// git history covers "undo". This is the one command
// whose file operation is simple enough to implement directly here rather
// than through a shared helper in internal/task.
func Delete(tasksDir string, req Request) error {
	if err := req.validate(); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	path := store.Path(tasksDir, req.ID)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return task.ErrTaskNotFound
		}
		return fmt.Errorf("stat task %s: %w", req.ID, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete task %s: %w", req.ID, err)
	}
	return nil
}
