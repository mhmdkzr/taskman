package approve

import (
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string) string {
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
			Review:         task.StageStatus{State: task.StagePending},
		},
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestApproveReview(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	comment := "LGTM"
	got, err := ApproveReview(dir, Request{ID: "abc", Comment: comment})
	if err != nil {
		t.Fatalf("ApproveReview: %v", err)
	}
	if got.Status.Review.State != task.StageDone {
		t.Fatalf("review.state = %v, want done", got.Status.Review.State)
	}
	if got.Status.Review.CompletedAt == nil {
		t.Fatal("review.completed_at is nil, want set")
	}
	if len(got.HumanReviews) != 1 {
		t.Fatalf("human_reviews length = %d, want 1", len(got.HumanReviews))
	}
	if !got.HumanReviews[0].Approved {
		t.Fatal("human_reviews[0].approved = false, want true")
	}
	if got.HumanReviews[0].Comment != comment {
		t.Fatalf("human_reviews[0].comment = %q, want %q", got.HumanReviews[0].Comment, comment)
	}
}

func TestApproveReviewTrunkCompletesTask(t *testing.T) {
	dir := t.TempDir()
	tk := task.Task{
		ID:         "abc",
		State:      task.StateStarted,
		Definition: "def",
		Status: task.Status{
			Definition:     task.StageStatus{State: task.StageDone},
			Specification:  task.StageStatus{State: task.StageDone},
			Implementation: task.StageStatus{State: task.StageDone},
			Verification:   task.StageStatus{State: task.StageDone},
			Review:         task.StageStatus{State: task.StagePending},
		},
		Git: task.Git{Worktree: dir, Branch: "main", Trunk: true},
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	got, err := ApproveReview(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("ApproveReview: %v", err)
	}
	if got.Status.Merge.State != task.StageDone {
		t.Errorf("merge.state = %v, want done - a trunk task has nothing left to merge", got.Status.Merge.State)
	}
	if got.State != task.StateCompleted {
		t.Errorf("state = %v, want completed", got.State)
	}
}

func TestApproveReviewTwice(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	if _, err := ApproveReview(dir, Request{ID: "abc", Comment: "LGTM"}); err != nil {
		t.Fatalf("first ApproveReview: %v", err)
	}
	if _, err := ApproveReview(dir, Request{ID: "abc", Comment: "LGTM again"}); err == nil {
		t.Fatal("second ApproveReview: want error, got nil")
	}
}
