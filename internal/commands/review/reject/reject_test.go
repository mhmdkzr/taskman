package reject

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, reviewPending bool) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateHumanReview,
		Definition: "def",
	}
	if !reviewPending {
		tk.State = task.StateFixHumanReviewFindings
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestRejectReview(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	reason := "needs more work"
	got, err := RejectReview(dir, Request{ID: "abc", Reason: reason})
	if err != nil {
		t.Fatalf("RejectReview: %v", err)
	}
	if got.State != task.StateFixHumanReviewFindings {
		t.Fatalf("state = %v, want fix_human_review_findings", got.State)
	}
	if len(got.HumanReviews) != 1 {
		t.Fatalf("human_reviews length = %d, want 1", len(got.HumanReviews))
	}
	if got.HumanReviews[0].Approved {
		t.Fatal("human_reviews[0].approved = true, want false")
	}
	if got.HumanReviews[0].Comment != reason {
		t.Fatalf("human_reviews[0].comment = %q, want %q", got.HumanReviews[0].Comment, reason)
	}
}

func TestRejectReviewRequiresPending(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	if _, err := RejectReview(dir, Request{ID: "abc", Reason: "reason"}); err == nil {
		t.Fatal("reject review when not pending: want error, got nil")
	}
}
