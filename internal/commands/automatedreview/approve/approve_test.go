package approve

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func TestApprove(t *testing.T) {
	dir := t.TempDir()
	value := task.Task{ID: "abc", State: task.StateAutomatedReview, Definition: "def"}
	if err := store.Write(dir, value); err != nil {
		t.Fatalf("write task: %v", err)
	}
	got, err := Approve(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if got.State != task.StateCommit {
		t.Fatalf("state = %q, want commit", got.State)
	}
}
