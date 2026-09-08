package task

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	run("add", "README.md")
	run("commit", "-q", "-m", "chore: init")
	return dir
}

func TestGitClientIsClean(t *testing.T) {
	dir := newTestRepo(t)
	git := NewGit(dir)
	ctx := context.Background()

	clean, err := git.IsClean(ctx)
	if err != nil {
		t.Fatalf("is clean: %v", err)
	}
	if !clean {
		t.Error("fresh repo: want clean, got dirty")
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	clean, err = git.IsClean(ctx)
	if err != nil {
		t.Fatalf("is clean: %v", err)
	}
	if clean {
		t.Error("dirty repo: want dirty, got clean")
	}
}

func TestGitClientCreateWorktreeAndReadCommit(t *testing.T) {
	dir := newTestRepo(t)
	git := NewGit(dir)
	ctx := context.Background()

	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	worktree, branch, err := git.CreateWorktree(ctx, worktreesDir, "abc")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}
	if branch != "task/abc" {
		t.Errorf("branch = %q, want task/abc", branch)
	}
	if _, err := os.Stat(filepath.Join(worktree, "README.md")); err != nil {
		t.Fatalf("worktree missing checked-out content: %v", err)
	}

	if err := os.WriteFile(filepath.Join(worktree, "NOTES.md"), []byte("notes\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	commitInWorktree := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = worktree
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	commitInWorktree("add", "NOTES.md")
	commitInWorktree("commit", "-q", "-m", "docs: add notes\n\nlonger body line")

	commit, err := git.ReadCommit(ctx, worktree, "")
	if err != nil {
		t.Fatalf("read commit: %v", err)
	}
	if commit.Type != "docs" {
		t.Errorf("type = %q, want docs", commit.Type)
	}
	if commit.Hash == "" {
		t.Error("hash is empty")
	}
	if commit.Message == "" {
		t.Error("message is empty")
	}
}

func TestGitClientReadCommitNoConventionalPrefix(t *testing.T) {
	dir := newTestRepo(t)
	git := NewGit(dir)
	ctx := context.Background()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("commit", "--allow-empty", "-q", "-m", "just a plain message, no prefix")

	commit, err := git.ReadCommit(ctx, dir, "HEAD")
	if err != nil {
		t.Fatalf("read commit: %v", err)
	}
	if commit.Type != "" {
		t.Errorf("type = %q, want empty (message has no conventional-commit prefix)", commit.Type)
	}
}

func TestCommitReadsRealCommit(t *testing.T) {
	dir := newTestRepo(t)
	git := NewGit(dir)
	ctx := context.Background()
	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	worktree, branch, err := git.CreateWorktree(ctx, worktreesDir, "abc")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	repo := NewRepo(t.TempDir())
	tk := Task{
		ID: "abc", State: StateStarted, Definition: "def",
		Status: Status{Verification: StageStatus{State: StageDone}},
		Git:    Git{Worktree: worktree, Branch: branch},
	}
	if err := repo.Create(tk); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := os.WriteFile(filepath.Join(worktree, "FIX.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = worktree
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("add", "FIX.md")
	run("commit", "-q", "-m", "fix: address review feedback")

	got, err := Commit(ctx, repo, git, "abc", CommitRequest{})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if got.Git.Commit == nil {
		t.Fatal("git.commit is nil")
	}
	if got.Git.Commit.Type != "fix" {
		t.Errorf("type = %q, want fix", got.Git.Commit.Type)
	}
	if got.Status.Review.State != StagePending {
		t.Errorf("review.state = %v, want pending", got.Status.Review.State)
	}
}

func TestCommitRequiresVerificationDone(t *testing.T) {
	dir := newTestRepo(t)
	git := NewGit(dir)
	ctx := context.Background()
	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	worktree, branch, err := git.CreateWorktree(ctx, worktreesDir, "abc")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	repo := NewRepo(t.TempDir())
	tk := Task{ID: "abc", State: StateStarted, Definition: "def", Git: Git{Worktree: worktree, Branch: branch}}
	if err := repo.Create(tk); err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := Commit(ctx, repo, git, "abc", CommitRequest{}); err == nil {
		t.Error("Commit before verification.state == done: want error, got nil")
	}
}
