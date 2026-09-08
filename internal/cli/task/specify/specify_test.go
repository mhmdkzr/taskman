package specify

import (
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, specDone bool) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateCreated,
		Definition: "def",
		Status: task.Status{
			Definition:     task.StageStatus{State: task.StageDone},
			Specification:  task.StageStatus{},
			Implementation: task.StageStatus{State: task.StagePending},
		},
	}
	if specDone {
		tk.Status.Specification = task.StageStatus{State: task.StageDone}
	} else {
		tk.Status.Specification = task.StageStatus{State: task.StagePending}
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestSpecify(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	req := Request{
		Result:   "this is the specification",
		DoneWhen: "when all tests pass",
	}
	got, err := Specify(dir, "abc", req)
	if err != nil {
		t.Fatalf("Specify: %v", err)
	}
	if got.Specification != req.Result {
		t.Fatalf("specification = %q, want %q", got.Specification, req.Result)
	}
	if got.DoneWhen != req.DoneWhen {
		t.Fatalf("done_when = %q, want %q", got.DoneWhen, req.DoneWhen)
	}
	if got.Status.Specification.State != task.StageDone {
		t.Fatalf("specification.state = %v, want done", got.Status.Specification.State)
	}
	if got.State != task.StateStarted {
		t.Fatalf("state = %v, want started", got.State)
	}
	if got.Status.Specification.CompletedAt == nil {
		t.Fatal("specification.completed_at not set")
	}
}

func TestSpecifyAlreadyDone(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	req := Request{
		Result:   "this is the specification",
		DoneWhen: "when all tests pass",
	}
	if _, err := Specify(dir, "abc", req); err == nil {
		t.Fatal("specify already-done specification: want error, got nil")
	}
}
