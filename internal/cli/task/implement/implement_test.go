package implement

import (
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

func newTestTaskDir(t *testing.T, id string, specDone bool) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateStarted,
		Definition: "def",
		Status: task.Status{
			Definition: task.StageStatus{State: task.StageDone},
		},
	}
	if specDone {
		tk.Status.Specification = task.StageStatus{State: task.StageDone}
	} else {
		tk.Status.Specification = task.StageStatus{State: task.StagePending}
	}
	tk.Status.Implementation = task.StageStatus{State: task.StagePending}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestImplement(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	got, err := Implement(dir, "abc")
	if err != nil {
		t.Fatalf("Implement: %v", err)
	}
	if got.Status.Implementation.State != task.StageDone {
		t.Fatalf("implementation.state = %v, want done", got.Status.Implementation.State)
	}
}

func TestImplementRequiresSpecification(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	if _, err := Implement(dir, "abc"); err == nil {
		t.Fatal("implement before specify: want error, got nil")
	}
}
