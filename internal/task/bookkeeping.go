package task

import (
	"context"
	"fmt"
	"path/filepath"
)

// RecordBookkeeping commits t's own file under tasksDir once it reaches a
// terminal state - see GitClient.CommitBookkeeping. verb ("completion" or
// "abandonment") reads into the commit message, e.g.
// `chore(task): Record completion of task "Fix Doc Drift" (ID: <id>)` -
// just `(ID: <id>)`, no quoted title, for an untitled task. Callers are
// responsible for calling this only once the task has actually gone
// terminal.
func RecordBookkeeping(ctx context.Context, git *GitClient, tasksDir string, t Task, verb string) error {
	path, err := filepath.Abs(taskPath(tasksDir, t.ID))
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
