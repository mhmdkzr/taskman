package delete

import (
	"errors"
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

func TestDeleteExistingTask(t *testing.T) {
	dir := t.TempDir()
	tk := task.Task{
		ID:         "abc",
		State:      task.StateStarted,
		Definition: "test task",
		Status: task.Status{
			Definition: task.StageStatus{State: task.StageDone},
		},
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	// Verify the task exists before deletion
	if _, err := task.ReadTask(dir, "abc"); err != nil {
		t.Fatalf("read task before delete: %v", err)
	}

	// Delete the task
	if err := Delete(dir, "abc"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify the task is gone
	_, err := task.ReadTask(dir, "abc")
	if !errors.Is(err, task.ErrTaskNotFound) {
		t.Fatalf("read task after delete: want ErrTaskNotFound, got %v", err)
	}
}

func TestDeleteNonExistentTask(t *testing.T) {
	dir := t.TempDir()
	if err := Delete(dir, "nonexistent"); !errors.Is(err, task.ErrTaskNotFound) {
		t.Fatalf("Delete: want ErrTaskNotFound, got %v", err)
	}
}
