// Package get owns the "get" command: its domain logic and CLI wiring.
package get

import (
	"fmt"

	"github.com/mhmdkzr/loop/internal/task"
)

// Get reads the current state of task id under tasksDir.
func Get(tasksDir, id string) (task.Task, error) {
	t, err := task.ReadTask(tasksDir, id)
	if err != nil {
		return task.Task{}, fmt.Errorf("read task: %w", err)
	}
	return t, nil
}
