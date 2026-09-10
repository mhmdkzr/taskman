package implement

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func newTestTaskDir(t *testing.T, id string, specDone bool) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateImplement,
		Definition: "def",
	}
	if specDone {
		tk.Specification = "spec"
		tk.DoneWhen = "done"
	} else {
		tk.State = task.StateSpecify
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestImplement(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	got, err := Implement(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("Implement: %v", err)
	}
	if got.State != task.StateVerify {
		t.Fatalf("state = %v, want verify", got.State)
	}
}

func TestImplementRequiresSpecification(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	if _, err := Implement(dir, Request{ID: "abc"}); err == nil {
		t.Fatal("implement before specify: want error, got nil")
	}
}
