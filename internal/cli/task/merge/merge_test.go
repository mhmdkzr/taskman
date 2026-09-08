package merge

import (
	"testing"
	"time"

	"github.com/mhmdkzr/loop/internal/task"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string, reviewDone bool) string {
	t.Helper()
	dir := t.TempDir()
	now := time.Now().UTC()
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
		Git: task.Git{
			Commit: &task.GitCommit{Hash: "abc123", Message: "x", Type: "feat", At: now},
		},
	}
	if reviewDone {
		tk.Status.Review = task.StageStatus{State: task.StageDone, CompletedAt: &now}
	} else {
		tk.Status.Review = task.StageStatus{State: task.StagePending}
	}
	tk.Status.Merge = task.StageStatus{State: task.StagePending}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestMerge(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	got, err := Merge(dir, "abc", Request{})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got.Status.Merge.State != task.StageDone {
		t.Fatalf("merge.state = %v, want done", got.Status.Merge.State)
	}
	if got.State != task.StateCompleted {
		t.Fatalf("state = %v, want completed", got.State)
	}
}

func TestMergeWithCommitOverride(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	newCommit := "def456"
	got, err := Merge(dir, "abc", Request{Commit: newCommit})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if got.Git.Commit == nil {
		t.Fatalf("git.commit is nil")
	}
	if got.Git.Commit.Hash != newCommit {
		t.Fatalf("commit hash = %v, want %v", got.Git.Commit.Hash, newCommit)
	}
}

func TestMergeRequiresReview(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	if _, err := Merge(dir, "abc", Request{}); err == nil {
		t.Fatal("merge before review done: want error, got nil")
	}
}
