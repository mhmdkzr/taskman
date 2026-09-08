package task

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/loop/internal/cli/task/review"
)

func TestMerge(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	readyForVerify(t, dir, id)
	if _, err := runCmd(t, dir, Verify(), id, "--check", "vet=ok"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if _, err := runCmd(t, dir, review.Record(), id, "--approved=true"); err != nil {
		t.Fatalf("review record: %v", err)
	}
	worktree := filepath.Join(dir, ".worktrees", id)
	gitCommitInWorktree(t, worktree, "feat: x")
	if _, err := runCmd(t, dir, Commit(), id); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := runCmd(t, dir, review.Approve(), id); err != nil {
		t.Fatalf("review approve: %v", err)
	}

	cmd := exec.Command("git", "merge", "--no-ff", "task/"+id, "-m", "merge")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git merge: %v: %s", err, out)
	}

	if _, err := runCmd(t, dir, Merge(), id, "--commit", "deadbeefcafe"); err != nil {
		t.Fatalf("merge: %v", err)
	}
	got := getJSON(t, dir, id)
	if got.State != "completed" {
		t.Fatalf("state = %v, want completed", got.State)
	}
	if got.Git.Commit == nil || got.Git.Commit.Hash != "deadbeefcafe" {
		t.Fatalf("git.commit = %+v, want overridden hash", got.Git.Commit)
	}
}

func TestMergeRequiresReviewDone(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Merge(), id); err == nil {
		t.Fatal("merge before review done: want error, got nil")
	}
}
