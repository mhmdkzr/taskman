package specify

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, specDone bool) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateSpecify,
		Definition: "def",
	}
	if specDone {
		tk.State = task.StateImplement
		tk.Specification = "existing spec"
		tk.DoneWhen = "existing criteria"
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestSpecify(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	req := Request{
		ID:       "abc",
		Result:   "this is the specification",
		DoneWhen: "when all tests pass",
	}
	got, err := Specify(dir, req)
	if err != nil {
		t.Fatalf("Specify: %v", err)
	}
	if got.Specification != req.Result {
		t.Fatalf("specification = %q, want %q", got.Specification, req.Result)
	}
	if got.DoneWhen != req.DoneWhen {
		t.Fatalf("done_when = %q, want %q", got.DoneWhen, req.DoneWhen)
	}
	if got.State != task.StateSpecificationReview {
		t.Fatalf("state = %v, want specification_review", got.State)
	}
}

func TestSpecifyAlreadyDone(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	req := Request{
		ID:       "abc",
		Result:   "this is the specification",
		DoneWhen: "when all tests pass",
	}
	if _, err := Specify(dir, req); err == nil {
		t.Fatal("specify already-done specification: want error, got nil")
	}
}
