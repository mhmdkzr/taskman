package git

import (
	"context"
	"os/exec"
	"testing"
)

// newTestRepo creates a fresh git repository in a temp dir with one commit,
// and returns its path.
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-q", "-m", "chore: init")
	return dir
}

func TestGitClientReadCommit(t *testing.T) {
	dir := newTestRepo(t)
	git := NewClient(dir)
	ctx := context.Background()

	commit, err := git.ReadCommit(ctx, dir, "")
	if err != nil {
		t.Fatalf("read commit: %v", err)
	}
	if commit.Hash == "" {
		t.Error("hash is empty")
	}
	if commit.Message == "" {
		t.Error("message is empty")
	}
	if commit.At.IsZero() {
		t.Error("at is zero")
	}
}

func TestGitClientReadCommitExplicitRef(t *testing.T) {
	dir := newTestRepo(t)
	git := NewClient(dir)
	ctx := context.Background()

	commit, err := git.ReadCommit(ctx, dir, "HEAD")
	if err != nil {
		t.Fatalf("read commit: %v", err)
	}
	if commit.Hash == "" {
		t.Error("hash is empty")
	}
}

func TestGitClientReadCommitRejectsUnknownRef(t *testing.T) {
	dir := newTestRepo(t)
	git := NewClient(dir)
	ctx := context.Background()

	if _, err := git.ReadCommit(ctx, dir, "not-a-real-ref"); err == nil {
		t.Fatal("read commit with unknown ref: want error, got nil")
	}
}
