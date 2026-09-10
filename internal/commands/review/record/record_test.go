package record

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
		tk.State = task.StateAutomatedReview
	}

	if blocked {
		tk.State = task.StateBlocked
		tk.Blocked = &task.Blocked{
			ResumeState: task.StateAutomatedReview,
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

func TestRecordReviewApproved(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)
	got, err := RecordReview(dir, Request{
		ID:       "abc",
		Approved: true,
		Findings: []task.Finding{},
	})
	if err != nil {
		t.Fatalf("RecordReview: %v", err)
	}
	if got.State != task.StateCommit {
		t.Fatalf("state = %v, want commit", got.State)
	}
	if len(got.Reviews) != 1 {
		t.Fatalf("reviews len = %d, want 1", len(got.Reviews))
	}
	if !got.Reviews[0].Approved {
		t.Fatal("review should be approved")
	}
}

func TestRecordReviewRejectedOnce(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)
	got, err := RecordReview(dir, Request{
		ID:       "abc",
		Approved: false,
		Findings: []task.Finding{
			{File: "main.go", Detail: "unused variable"},
		},
	})
	if err != nil {
		t.Fatalf("RecordReview: %v", err)
	}
	if got.State == task.StateBlocked {
		t.Fatal("first rejection should not block task")
	}
	if len(got.Reviews) != 1 {
		t.Fatalf("reviews len = %d, want 1", len(got.Reviews))
	}
	if got.Reviews[0].Approved {
		t.Fatal("review should not be approved")
	}
}

func TestRecordReviewRejectedTwice(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)

	// First rejection
	got1, err := RecordReview(dir, Request{
		ID:       "abc",
		Approved: false,
		Findings: []task.Finding{
			{File: "main.go", Detail: "unused variable"},
		},
	})
	if err != nil {
		t.Fatalf("first RecordReview: %v", err)
	}
	if got1.State == task.StateBlocked {
		t.Fatal("first rejection should not block task")
	}

	// Simulate the fix and passing verification that lead to the second review.
	got1, err = task.Apply(got1, task.VerificationReported{Verification: task.Verification{
		Checks: map[string]task.CheckResult{"tests": task.CheckOK}, CreatedAt: time.Now().UTC(),
	}})
	if err != nil {
		t.Fatalf("verify fix: %v", err)
	}
	if err := store.Write(dir, got1); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	// Second rejection
	got2, err := RecordReview(dir, Request{
		ID:       "abc",
		Approved: false,
		Findings: []task.Finding{
			{File: "main.go", Detail: "still has unused variable"},
		},
	})
	if err != nil {
		t.Fatalf("second RecordReview: %v", err)
	}
	if got2.State != task.StateBlocked {
		t.Fatalf("state = %v, want blocked", got2.State)
	}
	if got2.Blocked == nil {
		t.Fatal("blocked should not be nil")
	}
	if got2.Blocked.Stage != task.StageVerification {
		t.Fatalf("blocked.stage = %v, want verification", got2.Blocked.Stage)
	}
	if len(got2.Reviews) != 2 {
		t.Fatalf("reviews len = %d, want 2", len(got2.Reviews))
	}
}

func TestRecordReviewRequiresImplementation(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false, false)
	if _, err := RecordReview(dir, Request{ID: "abc", Approved: true}); err == nil {
		t.Fatal("record review before implementation done: want error, got nil")
	}
}

func TestRecordReviewRequiresNotBlocked(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, true)
	if _, err := RecordReview(dir, Request{ID: "abc", Approved: true}); err == nil {
		t.Fatal("record review on blocked task: want error, got nil")
	}
}
