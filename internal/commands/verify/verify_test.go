package verify

import (
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, implDone bool, blocked bool) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:            id,
		State:         task.StateImplement,
		Definition:    "def",
		Specification: "spec",
		DoneWhen:      "done",
	}
	if implDone {
		tk.State = task.StateVerify
	}
	if blocked {
		tk.State = task.StateBlocked
		tk.Blocked = &task.Blocked{
			ResumeState: task.StateVerify,
			Stage:       task.StageVerification,
			Reason:      "test block",
			At:          time.Now().UTC(),
		}
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestVerifyAppends(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)

	// First verification with a failing check
	got, err := Verify(dir, Request{
		ID: "abc",
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
	got, err = Verify(dir, Request{
		ID: "abc",
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
	if _, err := Verify(dir, Request{
		ID: "abc",
		Checks: map[string]task.CheckResult{
			"vet": task.CheckOK,
		},
	}); err == nil {
		t.Fatal("verify before implementation done: want error, got nil")
	}
}

func TestVerifyRejectsBlockedTask(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, true)
	if _, err := Verify(dir, Request{
		ID: "abc",
		Checks: map[string]task.CheckResult{
			"vet": task.CheckOK,
		},
	}); err == nil {
		t.Fatal("verify on blocked task: want error, got nil")
	}
}
