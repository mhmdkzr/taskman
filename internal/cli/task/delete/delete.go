// Package delete owns the "delete" command: its domain logic and CLI
// wiring.
package delete

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Delete removes task id's file outright under tasksDir - no soft-delete,
// git history covers "undo" (design.md §3/§6). This is the one command
// whose file operation is simple enough to implement directly here rather
// than through a shared helper in internal/task.
func Delete(tasksDir, id string) error {
	path := filepath.Join(tasksDir, id+".yaml")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return task.ErrTaskNotFound
		}
		return fmt.Errorf("stat task %s: %w", id, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete task %s: %w", id, err)
	}
	return nil
}
