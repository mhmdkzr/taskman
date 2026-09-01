package codebase

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// CloneOptions configures Clone.
type CloneOptions struct {
	// Branch is created and checked out in the clone, branching from the
	// source repository's current HEAD. Empty checks out whatever branch (or
	// detached HEAD) the clone starts on, same as a plain `git clone` would.
	Branch string
}

// Clone creates an isolated local clone of the repository containing
// sourceDir at destDir — its own worktree, object database, and (with
// opts.Branch set) a fresh branch — so a task's tools can never touch the
// source repository no matter what they do to the result. destDir must not
// already exist.
func Clone(sourceDir, destDir string, opts CloneOptions) (Repository, error) {
	src, err := Open(sourceDir)
	if err != nil {
		return Repository{}, err
	}
	root, err := src.root()
	if err != nil {
		return Repository{}, err
	}

	repo, err := git.PlainClone(destDir, false, &git.CloneOptions{URL: root})
	if err != nil {
		return Repository{}, fmt.Errorf("clone %s: %w", root, err)
	}
	clone := Repository{r: repo}

	if opts.Branch != "" {
		wt, err := repo.Worktree()
		if err != nil {
			return Repository{}, fmt.Errorf("worktree: %w", err)
		}
		if err := wt.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(opts.Branch),
			Create: true,
		}); err != nil {
			return Repository{}, fmt.Errorf("checkout %s: %w", opts.Branch, err)
		}
	}

	return clone, nil
}

// Remove deletes this repository's entire worktree directory — the clone
// Clone created, not the source repository it was cloned from. A reviewed
// task's commit lives in the source's git history via a merge/push regardless
// of whether the clone that produced it still exists on disk.
func (r Repository) Remove() error {
	root, err := r.root()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(root); err != nil {
		return fmt.Errorf("remove %s: %w", root, err)
	}
	return nil
}

// IsClean reports whether the worktree has no staged or unstaged changes.
func (r Repository) IsClean() (bool, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return false, fmt.Errorf("worktree: %w", err)
	}
	status, err := wt.Status()
	if err != nil {
		return false, fmt.Errorf("status: %w", err)
	}
	return status.IsClean(), nil
}
