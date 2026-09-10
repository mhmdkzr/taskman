package get

import (
	"errors"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func newTestTaskDir(t *testing.T, id string) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateImplement,
		Definition: "test definition",
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestGet(t *testing.T) {
	dir := newTestTaskDir(t, "test-id")
	got, err := Get(dir, Request{ID: "test-id"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "test-id" {
		t.Fatalf("ID = %s, want test-id", got.ID)
	}
	if got.Definition != "test definition" {
		t.Fatalf("Definition = %s, want test definition", got.Definition)
	}
	if got.State != task.StateImplement {
		t.Fatalf("State = %v, want implement", got.State)
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := Get(dir, Request{ID: "non-existent"})
	if err == nil {
		t.Fatal("Get non-existent: want error, got nil")
	}
	if !errors.Is(err, task.ErrTaskNotFound) {
		t.Fatalf("Get non-existent: err = %v, want ErrTaskNotFound", err)
	}
}

func TestGetEmptyID(t *testing.T) {
	dir := t.TempDir()
	if _, err := Get(dir, Request{}); err == nil {
		t.Fatal("Get with empty id: want error, got nil")
	}
}
