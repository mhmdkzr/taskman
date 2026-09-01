package codebase

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// newTestRepo creates a scratch git repo with one commit and returns its
// directory.
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	sig := &object.Signature{Name: "Test", Email: "test@example.com"}
	if _, err := wt.Commit("initial commit", &git.CommitOptions{Author: sig}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return dir
}

// TestCloneIsolatesWorktree verifies Clone produces an independent checkout
// on the requested branch, and that edits in the clone never touch the
// source repository — the whole point of cloning per task.
func TestCloneIsolatesWorktree(t *testing.T) {
	source := newTestRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	clone, err := Clone(source, dest, CloneOptions{Branch: "task/example"})
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, "README.md")); err != nil {
		t.Errorf("cloned worktree missing README.md: %v", err)
	}

	branch, err := clone.Branch()
	if err != nil {
		t.Fatalf("Branch: %v", err)
	}
	if branch.Name != "task/example" {
		t.Errorf("branch = %q, want task/example", branch.Name)
	}

	clean, err := clone.IsClean()
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean {
		t.Error("fresh clone should be clean")
	}

	if err := os.WriteFile(filepath.Join(dest, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	clean, err = clone.IsClean()
	if err != nil {
		t.Fatalf("IsClean after edit: %v", err)
	}
	if clean {
		t.Error("clone should be dirty after edit")
	}

	sourceContent, err := os.ReadFile(filepath.Join(source, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile source: %v", err)
	}
	if string(sourceContent) != "hello\n" {
		t.Errorf("editing the clone changed the source: %q", sourceContent)
	}

	if err := clone.Remove(); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("clone dir still exists after Remove (err=%v)", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Errorf("Remove must not touch the source repo: %v", err)
	}
}

// TestCloneRejectsExistingRepo verifies Clone surfaces go-git's error instead
// of silently clobbering a destination that's already a git repository (e.g.
// a stale leftover from a previous, uncleaned-up task run).
func TestCloneRejectsExistingRepo(t *testing.T) {
	source := newTestRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	if _, err := Clone(source, dest, CloneOptions{}); err != nil {
		t.Fatalf("first Clone: %v", err)
	}
	if _, err := Clone(source, dest, CloneOptions{}); err == nil {
		t.Error("Clone into an existing repo: want error")
	}
}
