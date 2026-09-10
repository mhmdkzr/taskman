package git

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// RecordBookkeeping commits a terminal task's own persisted file.
func RecordBookkeeping(ctx context.Context, git *Client, tasksDir string, t task.Task, verb string) error {
	path, err := filepath.Abs(store.Path(tasksDir, t.ID))
	if err != nil {
		return fmt.Errorf("resolve task file path: %w", err)
	}
	subject := fmt.Sprintf("(ID: %s)", t.ID)
	if t.Title != "" {
		subject = fmt.Sprintf("%q %s", t.Title, subject)
	}
	message := fmt.Sprintf("chore(task): Record %s of task %s", verb, subject)
	if err := git.CommitBookkeeping(ctx, path, message); err != nil {
		return fmt.Errorf("commit bookkeeping: %w", err)
	}
	return nil
}
