package reject

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func TestReject(t *testing.T) {
	dir := t.TempDir()
	value := task.Task{ID: "abc", State: task.StateAutomatedReview, Definition: "def"}
	if err := store.Write(dir, value); err != nil {
		t.Fatalf("write task: %v", err)
	}
	got, err := Reject(dir, Request{ID: "abc", Findings: []task.Finding{{File: "main.go", Detail: "unused variable"}}})
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if got.State != task.StateFixAutomatedReviewFindings {
		t.Fatalf("state = %q, want fix_automated_review_findings", got.State)
	}
}

func TestRejectRequiresFinding(t *testing.T) {
	if _, err := Reject(t.TempDir(), Request{ID: "abc"}); err == nil {
		t.Fatal("Reject without findings: want error")
	}
}
