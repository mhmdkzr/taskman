package escalate

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, state task.State) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      state,
		Definition: "def",
		Status: task.Status{
			Definition: task.StageStatus{State: task.StageDone},
		},
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestEscalate(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateStarted)
	got, err := Escalate(dir, "abc", Request{
		Stage:  "implementation",
		Reason: "agent gave up",
	})
	if err != nil {
		t.Fatalf("Escalate: %v", err)
	}
	if got.State != task.StateBlocked {
		t.Fatalf("state = %v, want blocked", got.State)
	}
	if got.Blocked == nil {
		t.Fatal("Blocked is nil, want non-nil")
	}
	if got.Blocked.Stage != "implementation" {
		t.Fatalf("blocked.stage = %v, want implementation", got.Blocked.Stage)
	}
	if got.Blocked.Reason != "agent gave up" {
		t.Fatalf("blocked.reason = %v, want 'agent gave up'", got.Blocked.Reason)
	}
}

func TestEscalateAlreadyCompleted(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateCompleted)
	if _, err := Escalate(dir, "abc", Request{
		Stage:  "implementation",
		Reason: "agent gave up",
	}); err == nil {
		t.Fatal("escalate completed task: want error, got nil")
	}
}

func TestEscalateAlreadyFailed(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateFailed)
	if _, err := Escalate(dir, "abc", Request{
		Stage:  "implementation",
		Reason: "agent gave up",
	}); err == nil {
		t.Fatal("escalate failed task: want error, got nil")
	}
}
