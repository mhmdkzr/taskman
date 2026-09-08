package reject

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, reviewPending bool) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateStarted,
		Definition: "def",
		Status: task.Status{
			Definition:     task.StageStatus{State: task.StageDone},
			Specification:  task.StageStatus{State: task.StageDone},
			Implementation: task.StageStatus{State: task.StageDone},
			Verification:   task.StageStatus{State: task.StageDone},
		},
	}
	if reviewPending {
		tk.Status.Review = task.StageStatus{State: task.StagePending}
	} else {
		tk.Status.Review = task.StageStatus{State: task.StageInProgress}
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
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
	if got.Status.Review.State != task.StageInProgress {
		t.Fatalf("review.state = %v, want in_progress", got.Status.Review.State)
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
