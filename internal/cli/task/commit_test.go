package task

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/loop/internal/cli/task/review"
)

func gitCommitInWorktree(t *testing.T, worktree, message string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = worktree
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(worktree, "CHANGE.md"), []byte("change\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "CHANGE.md")
	run("commit", "-q", "-m", message)
}

func TestCommit(t *testing.T) {
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
	got := getJSON(t, dir, id)
	if got.Git.Commit == nil || got.Git.Commit.Type != "feat" {
		t.Fatalf("git.commit = %+v", got.Git.Commit)
	}
}

func TestCommitRequiresVerificationDone(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Commit(), id); err == nil {
		t.Fatal("commit before verification done: want error, got nil")
	}
}
