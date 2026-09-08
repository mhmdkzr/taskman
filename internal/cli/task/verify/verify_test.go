package verify

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, implDone bool, blocked bool) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateStarted,
		Definition: "def",
		Status: task.Status{
			Definition:     task.StageStatus{State: task.StageDone},
			Specification:  task.StageStatus{State: task.StageDone},
			Implementation: task.StageStatus{State: task.StagePending},
		},
	}
	if implDone {
		tk.Status.Implementation = task.StageStatus{State: task.StageDone}
	}
	if blocked {
		tk.State = task.StateBlocked
		tk.Blocked = &task.Blocked{
			Stage:  task.StageVerification,
			Reason: "test block",
			At:     task.Now(),
		}
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestVerifyAppends(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)

	// First verification with a failing check
	got, err := Verify(dir, "abc", Request{
		Checks: map[string]task.CheckResult{
			"vet": task.CheckError,
		},
		Output: "vet failed",
	})
	if err != nil {
		t.Fatalf("Verify first: %v", err)
	}
	if len(got.Verifications) != 1 {
		t.Fatalf("after first verify: len(verifications) = %d, want 1", len(got.Verifications))
	}
	if got.Verifications[0].Passed() {
		t.Fatalf("first verification should not have passed")
	}

	// Second verification with all passing checks
	got, err = Verify(dir, "abc", Request{
		Checks: map[string]task.CheckResult{
			"vet":  task.CheckOK,
			"test": task.CheckOK,
		},
		Output: "all passed",
	})
	if err != nil {
		t.Fatalf("Verify second: %v", err)
	}
	if len(got.Verifications) != 2 {
		t.Fatalf("after second verify: len(verifications) = %d, want 2", len(got.Verifications))
	}
	if !got.Verifications[1].Passed() {
		t.Fatalf("second verification should have passed")
	}
}

func TestVerifyRequiresImplementationDone(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false, false)
	if _, err := Verify(dir, "abc", Request{
		Checks: map[string]task.CheckResult{
			"vet": task.CheckOK,
		},
	}); err == nil {
		t.Fatal("verify before implementation done: want error, got nil")
	}
}

func TestVerifyRejectsBlockedTask(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, true)
	if _, err := Verify(dir, "abc", Request{
		Checks: map[string]task.CheckResult{
			"vet": task.CheckOK,
		},
	}); err == nil {
		t.Fatal("verify on blocked task: want error, got nil")
	}
}
