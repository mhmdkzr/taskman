package approve

import (
	"os"
	"os/exec"
	"path/filepath"
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
	got, err := ApproveReview(t.Context(), dir, task.NewGit(dir), Request{ID: "abc", Comment: comment})
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

// newTestGitTaskDir is like newTestTaskDir, but dir is also a real git repo
// with tk committed - needed whenever ApproveReview is expected to complete
// the task (a trunk task), since that also commits the task's own file.
func newTestGitTaskDir(t *testing.T, tk task.Task) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.lock\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	git("add", ".gitignore")
	git("commit", "-q", "-m", "chore: init")
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	git("add", tk.ID+".yaml")
	git("commit", "-q", "-m", "chore: seed task")
	return dir
}

func TestApproveReviewTrunkCompletesTask(t *testing.T) {
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
		Git: task.Git{Branch: "main", Trunk: true},
	}
	dir := newTestGitTaskDir(t, tk)
	git := task.NewGit(dir)

	got, err := ApproveReview(t.Context(), dir, git, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("ApproveReview: %v", err)
	}
	if got.Status.Merge.State != task.StageDone {
		t.Errorf("merge.state = %v, want done - a trunk task has nothing left to merge", got.Status.Merge.State)
	}
	if got.State != task.StateCompleted {
		t.Errorf("state = %v, want completed", got.State)
	}
	clean, err := git.IsClean(t.Context())
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean {
		t.Fatal("working tree not clean after ApproveReview completed the task: want the task file committed")
	}
}

func TestApproveReviewTwice(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	git := task.NewGit(dir)
	if _, err := ApproveReview(t.Context(), dir, git, Request{ID: "abc", Comment: "LGTM"}); err != nil {
		t.Fatalf("first ApproveReview: %v", err)
	}
	if _, err := ApproveReview(t.Context(), dir, git, Request{ID: "abc", Comment: "LGTM again"}); err == nil {
		t.Fatal("second ApproveReview: want error, got nil")
	}
}
