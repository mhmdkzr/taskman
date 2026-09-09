package task

import (
	"bytes"
	"context"
	"errors"
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

// CreateWorktree runs `git worktree add <worktreesDir>/<id> -b task/<slug>`
// against the repo at g.dir, and returns the resulting worktree path and
// branch name. The worktree directory is keyed by the full id (unique even
// across tasks sharing a slug); the branch name uses only slug, per design.md
// §3 ("`task/<slug>` becomes the git branch name").
func (g *GitClient) CreateWorktree(ctx context.Context, worktreesDir, id, slug string) (string, string, error) {
	worktree := filepath.Join(worktreesDir, id)
	branch := "task/" + slug
	if _, err := g.run(ctx, g.dir, "worktree", "add", worktree, "-b", branch); err != nil {
		return "", "", err
	}
	return worktree, branch, nil
}

// UseTrunk returns the repo root at g.dir and the name of its currently
// checked-out branch, for `create --trunk` (design.md §5): it skips
// CreateWorktree entirely and works the task directly on the caller's
// current branch instead of an isolated worktree/branch pair.
func (g *GitClient) UseTrunk(ctx context.Context) (string, string, error) {
	out, err := g.run(ctx, g.dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "HEAD" {
		return "", "", fmt.Errorf("use trunk: repository is in detached HEAD state")
	}
	return g.dir, branch, nil
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

// CommitBookkeeping stages path and commits it with message - taskman's own
// commit, the one exception beyond §5's worktree creation. A terminal
// task's own file keeps changing through review and merge, so it can never
// ride along with the code commit that finished it (design.md §"final
// bookkeeping commit"); rather than dispatch that as another round trip,
// taskman makes this one commit itself. A no-op if path has nothing to
// commit (already committed, or gitignored).
func (g *GitClient) CommitBookkeeping(ctx context.Context, path, message string) error {
	if _, err := g.run(ctx, g.dir, "add", "--", path); err != nil {
		return err
	}
	staged, err := g.hasStagedChanges(ctx)
	if err != nil {
		return fmt.Errorf("check staged changes: %w", err)
	}
	if !staged {
		return nil
	}
	_, err = g.run(ctx, g.dir, "commit", "-m", message)
	return err
}

// hasStagedChanges reports whether the index has anything staged, via `git
// diff --cached --quiet`'s exit code (0 clean, 1 staged changes - anything
// else is a real error, not an answer).
func (g *GitClient) hasStagedChanges(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	cmd.Dir = g.dir
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("git diff --cached --quiet: %w", err)
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
