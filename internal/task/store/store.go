package store

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"

	"github.com/mhmdkzr/taskman/internal/task"
)

const filePerm = 0o600

// Path returns the YAML path for id under tasksDir.
func Path(tasksDir, id string) string { return filepath.Join(tasksDir, id+".yaml") }

// Read loads and validates a task document.
func Read(tasksDir, id string) (task.Task, error) {
	data, err := os.ReadFile(Path(tasksDir, id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return task.Task{}, task.ErrTaskNotFound
		}
		return task.Task{}, fmt.Errorf("read task %s: %w", id, err)
	}
	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return task.Task{}, fmt.Errorf("parse task %s: %w", id, err)
	}
	t, err := doc.taskValue()
	if err != nil {
		return task.Task{}, fmt.Errorf("parse task %s: %w", id, err)
	}
	return t, nil
}

// Write atomically writes a validated task document.
func Write(tasksDir string, t task.Task) error {
	if err := task.Validate(t); err != nil {
		return fmt.Errorf("validate task: %w", err)
	}
	data, err := yaml.Marshal(newDocument(t))
	if err != nil {
		return fmt.Errorf("marshal task %s: %w", t.ID, err)
	}
	path := Path(tasksDir, t.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, filePerm); err != nil {
		return fmt.Errorf("write task %s: %w", t.ID, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename task %s: %w", t.ID, err)
	}
	return nil
}

// Update performs a locked, atomic, value-oriented read-modify-write cycle.
func Update(tasksDir, id string, update func(task.Task) (task.Task, error)) (task.Task, error) {
	lock := flock.New(Path(tasksDir, id) + ".lock")
	if err := lock.Lock(); err != nil {
		return task.Task{}, fmt.Errorf("lock task %s: %w", id, err)
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			slog.Error("unlock task file", "id", id, "error", err)
		}
	}()

	current, err := Read(tasksDir, id)
	if err != nil {
		return task.Task{}, err
	}
	updated, err := update(current)
	if err != nil {
		return task.Task{}, err
	}
	if updated.ID != current.ID {
		return task.Task{}, fmt.Errorf("update task %s: id cannot change", id)
	}
	if err := Write(tasksDir, updated); err != nil {
		return task.Task{}, err
	}
	return updated, nil
}
