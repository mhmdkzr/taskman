package task

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GitClient wraps the handful of git operations taskman performs itself -
// design.md §5 and §6's "task commit" row. Every other git operation (build
// checks, commits, merges) is caller-executed and only reported to taskman.
type GitClient struct {
	dir string
}

// NewGit returns a GitClient rooted at dir (design.md §1's --git-dir).
func NewGit(dir string) *GitClient {
	return &GitClient{dir: dir}
}

// IsClean reports whether the working tree has no uncommitted changes
// (`git status --porcelain` is empty) - the precondition for task create.
func (g *GitClient) IsClean(ctx context.Context) (bool, error) {
	out, err := g.run(ctx, g.dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// CreateWorktree runs `git worktree add <worktreesDir>/<id> -b task/<id>`
// against the repo at g.dir, and returns the resulting worktree path and
// branch name.
func (g *GitClient) CreateWorktree(ctx context.Context, worktreesDir, id string) (string, string, error) {
	worktree := filepath.Join(worktreesDir, id)
	branch := "task/" + id
	if _, err := g.run(ctx, g.dir, "worktree", "add", worktree, "-b", branch); err != nil {
		return "", "", err
	}
	return worktree, branch, nil
}

// conventionalType matches a leading conventional-commit type prefix, e.g.
// "feat:", "fix(scope):", "feat!:".
var conventionalType = regexp.MustCompile(`^([a-zA-Z]+)(\([^)]*\))?!?:\s`)

// ReadCommit reads the commit at ref (typically "HEAD", or a hash) out of
// the worktree at worktreeDir via `git log`, rather than trusting a
// caller's own report of it - design.md §6 "When the commit happens". It
// parses a leading conventional-commit type prefix out of the message when
// present.
func (g *GitClient) ReadCommit(ctx context.Context, worktreeDir, ref string) (GitCommit, error) {
	if ref == "" {
		ref = "HEAD"
	}
	hash, err := g.run(ctx, worktreeDir, "log", "-1", "--format=%H", ref)
	if err != nil {
		return GitCommit{}, err
	}
	message, err := g.run(ctx, worktreeDir, "log", "-1", "--format=%B", ref)
	if err != nil {
		return GitCommit{}, err
	}
	hash = strings.TrimSpace(hash)
	message = strings.TrimRight(message, "\n")

	var commitType string
	if m := conventionalType.FindStringSubmatch(message); m != nil {
		commitType = m[1]
	}
	return GitCommit{Type: commitType, Message: message, Hash: hash}, nil
}

func (g *GitClient) run(ctx context.Context, dir string, args ...string) (string, error) {
	//nolint:gosec // running our own git wrapper; args are our own paths/ids, not attacker input
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
