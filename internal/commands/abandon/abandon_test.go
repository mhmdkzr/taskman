package abandon

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
			Definition:     task.StageStatus{State: task.StageDone},
			Specification:  task.StageStatus{State: task.StageDone},
			Implementation: task.StageStatus{State: task.StageDone},
		},
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestAbandon(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateStarted)
	got, err := Abandon(dir, Request{ID: "abc", Reason: "no longer needed"})
	if err != nil {
		t.Fatalf("Abandon: %v", err)
	}
	if got.State != task.StateFailed {
		t.Fatalf("state = %v, want %v", got.State, task.StateFailed)
	}
	if got.FailureReason != "no longer needed" {
		t.Fatalf("failure_reason = %q, want %q", got.FailureReason, "no longer needed")
	}
}

func TestAbandonPreventCompleted(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateCompleted)
	if _, err := Abandon(dir, Request{ID: "abc", Reason: "test reason"}); err == nil {
		t.Fatal("abandon completed task: want error, got nil")
	}
}

func TestAbandonFailedIdempotent(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateFailed)
	got, err := Abandon(dir, Request{ID: "abc", Reason: "updated reason"})
	if err != nil {
		t.Fatalf("abandon failed task again: %v", err)
	}
	if got.State != task.StateFailed {
		t.Fatalf("state = %v, want %v", got.State, task.StateFailed)
	}
	if got.FailureReason != "updated reason" {
		t.Fatalf("failure_reason = %q, want %q", got.FailureReason, "updated reason")
	}
}
