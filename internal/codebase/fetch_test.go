package codebase

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// TestFetchBranchMakesCloneCommitReachableInSource is a regression test: a
// live pipeline run produced a real commit inside an isolated Clone, then
// removed that clone on success without ever landing the commit anywhere
// else — the commit was gone the moment the clone directory was deleted.
// FetchBranch is the fix: the commit must be reachable in the source
// repository before the clone can safely be removed.
func TestFetchBranchMakesCloneCommitReachableInSource(t *testing.T) {
	source := newTestRepo(t)
	cloneDir := filepath.Join(t.TempDir(), "clone")

	clone, err := Clone(source, cloneDir, CloneOptions{Branch: "task/example"})
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	// Make a commit in the clone, as a task's work would.
	root, err := clone.Root()
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	result, err := clone.Commit(
		NewCommitMessage("test: change README"),
		AuthorSignature{Name: "Test", Email: "test@example.com"},
	)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	sourceRepo, err := Open(source)
	if err != nil {
		t.Fatalf("Open source: %v", err)
	}
	if err := sourceRepo.FetchBranch(cloneDir, "task/example"); err != nil {
		t.Fatalf("FetchBranch: %v", err)
	}

	// Remove the clone entirely, as RunTask does on success — the commit
	// must still be reachable from the source repo afterward.
	if err := clone.Remove(); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(cloneDir); !os.IsNotExist(err) {
		t.Fatalf("clone dir should be gone, err=%v", err)
	}

	rawSource, err := git.PlainOpen(source)
	if err != nil {
		t.Fatalf("PlainOpen source: %v", err)
	}
	hash := plumbing.NewHash(string(result.Hash))
	commit, err := rawSource.CommitObject(hash)
	if err != nil {
		t.Fatalf("commit %s not reachable in source after clone removal: %v", result.Hash, err)
	}
	if commit.Message != "test: change README" {
		t.Errorf("commit message = %q", commit.Message)
	}

	branchRef, err := rawSource.Reference(plumbing.NewBranchReferenceName("task/example"), true)
	if err != nil {
		t.Fatalf("branch task/example not found in source: %v", err)
	}
	if branchRef.Hash() != hash {
		t.Errorf("branch task/example = %s, want %s", branchRef.Hash(), hash)
	}
}
