package task

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
)

const filePerm = 0o600

// document is the on-disk shape: a task file is { task: <Task> }.
type document struct {
	Task Task `yaml:"task"`
}

func taskPath(tasksDir, id string) string {
	return filepath.Join(tasksDir, id+".yaml")
}

// ReadTask reads the current state of task id from tasksDir.
func ReadTask(tasksDir, id string) (Task, error) {
	data, err := os.ReadFile(taskPath(tasksDir, id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("read task %s: %w", id, err)
	}
	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return Task{}, fmt.Errorf("parse task %s: %w", id, err)
	}
	return doc.Task, nil
}

// WriteTaskFile marshals t and writes it to tasksDir via a temp-file-then-
// rename cycle, atomic on the same filesystem.
func WriteTaskFile(tasksDir string, t Task) error {
	data, err := yaml.Marshal(document{Task: t})
	if err != nil {
		return fmt.Errorf("marshal task %s: %w", t.ID, err)
	}
	path := taskPath(tasksDir, t.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, filePerm); err != nil {
		return fmt.Errorf("write task %s: %w", t.ID, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename task %s: %w", t.ID, err)
	}
	return nil
}

// MutateTask locks task id's file, loads it, calls fn to validate and apply
// a transition, and writes the result back - the read-modify-write-rename
// cycle every command goes through. fn returning an error aborts the
// mutation: nothing is written, and that error is returned as MutateTask's
// own.
func MutateTask(tasksDir, id string, fn func(t *Task) error) (Task, error) {
	// The lock is taken on a separate, stable file - never on the task
	// file itself. WriteTaskFile's atomic rename replaces the task file's
	// inode on every write; a lock held on a path survives only as long as
	// the inode it was opened against, so locking the renamed file
	// directly would let a concurrent process's fresh open() onto the new
	// inode acquire an independent, non-contending lock - silently
	// defeating cross-process exclusion the moment a write ever succeeded.
	lock := flock.New(taskPath(tasksDir, id) + ".lock")
	if err := lock.Lock(); err != nil {
		return Task{}, fmt.Errorf("lock task %s: %w", id, err)
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			slog.Error("unlock task file", "id", id, "error", err)
		}
	}()

	t, err := ReadTask(tasksDir, id)
	if err != nil {
		return Task{}, err
	}
	if err := fn(&t); err != nil {
		return Task{}, err
	}
	if err := WriteTaskFile(tasksDir, t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// now is a seam for tests; production code always uses time.Now().
var now = func() time.Time { return time.Now().UTC() }

// Now returns the current time - a seam so callers outside this package can
// stamp CompletedAt fields consistently with internal/task's own test seam.
func Now() time.Time { return now() }
