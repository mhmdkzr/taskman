package delete

import (
	"errors"
	"os"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func TestDeleteExistingTask(t *testing.T) {
	dir := t.TempDir()
	tk := task.Task{
		ID:         "abc",
		State:      task.StateImplement,
		Definition: "test task",
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	lockPath := store.Path(dir, tk.ID) + ".lock"
	if err := os.WriteFile(lockPath, nil, 0o600); err != nil {
		t.Fatalf("write lock file: %v", err)
	}

	// Verify the task exists before deletion
	if _, err := store.Read(dir, "abc"); err != nil {
		t.Fatalf("read task before delete: %v", err)
	}

	// Delete the task
	if err := Delete(dir, Request{ID: "abc"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify the task is gone
	_, err := store.Read(dir, "abc")
	if !errors.Is(err, task.ErrTaskNotFound) {
		t.Fatalf("read task after delete: want ErrTaskNotFound, got %v", err)
	}
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat lock after delete: want not found, got %v", err)
	}
}

func TestDeleteNonExistentTask(t *testing.T) {
	dir := t.TempDir()
	if err := Delete(dir, Request{ID: "nonexistent"}); !errors.Is(err, task.ErrTaskNotFound) {
		t.Fatalf("Delete: want ErrTaskNotFound, got %v", err)
	}
}

func TestDeleteExistingTaskWithoutLock(t *testing.T) {
	dir := t.TempDir()
	tk := task.Task{ID: "abc", State: task.StateImplement, Definition: "test task"}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	if err := Delete(dir, Request{ID: tk.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
