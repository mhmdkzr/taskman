package codebase

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Repository struct {
	r *git.Repository
}

// Open opens the git repository containing dir (walking up to find .git,
// like git itself does) and returns a Repository wrapping it. This is the
// normal entry point: the orchestrator calls Open once per task worktree,
// and every codebase/tools/codebase operation on the result is scoped to
// that worktree.
func Open(dir string) (Repository, error) {
	repo, err := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return Repository{}, fmt.Errorf("open git repo: %w", err)
	}
	return Repository{r: repo}, nil
}

// Root returns the worktree's filesystem root — the directory every
// operation in this package (go test/build/vet, file reads/writes, glob,
// grep) is scoped to.
func (r Repository) Root() (string, error) {
	return r.root()
}

// root is the unexported implementation Root and resolvePath share.
func (r Repository) root() (string, error) {
	wt, err := r.r.Worktree()
	if err != nil {
		return "", fmt.Errorf("worktree: %w", err)
	}
	return wt.Filesystem.Root(), nil
}

// resolvePath joins rel against the worktree root and rejects anything that
// would escape it (a ".." climb or an absolute path elsewhere on the host).
// Every file-touching operation in this package goes through this instead of
// using a caller-supplied path directly, so a task's tools cannot read or
// write outside its own worktree no matter what path an agent passes in.
func (r Repository) resolvePath(rel string) (string, error) {
	if rel == "" {
		return "", fmt.Errorf("path is required")
	}
	root, err := r.root()
	if err != nil {
		return "", err
	}
	root = filepath.Clean(root)

	var full string
	if filepath.IsAbs(rel) {
		full = filepath.Clean(rel)
	} else {
		full = filepath.Clean(filepath.Join(root, rel))
	}

	if full != root && !strings.HasPrefix(full, root+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes the worktree root %q", rel, root)
	}
	return full, nil
}
