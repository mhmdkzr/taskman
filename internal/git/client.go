// Package gitclient provides taskman's imperative Git adapter.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Client wraps the Git operations taskman performs itself.
type Client struct {
	dir string
}

func NewClient(dir string) *Client {
	return &Client{dir: dir}
}

func (g *Client) IsClean(ctx context.Context) (bool, error) {
	out, err := g.run(ctx, g.dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

func (g *Client) CreateWorktree(ctx context.Context, worktreesDir, id, slug string) (string, string, error) {
	worktree := filepath.Join(worktreesDir, id)
	branch := "task/" + slug
	if _, err := g.run(ctx, g.dir, "worktree", "add", worktree, "-b", branch); err != nil {
		return "", "", err
	}
	return worktree, branch, nil
}

func (g *Client) UseTrunk(ctx context.Context) (string, string, error) {
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

var conventionalType = regexp.MustCompile(`^([a-zA-Z]+)(\([^)]*\))?!?:\s`)

func (g *Client) ReadCommit(ctx context.Context, worktreeDir, ref string) (task.GitCommit, error) {
	if ref == "" {
		ref = "HEAD"
	}
	hash, err := g.run(ctx, worktreeDir, "log", "-1", "--format=%H", ref)
	if err != nil {
		return task.GitCommit{}, err
	}
	message, err := g.run(ctx, worktreeDir, "log", "-1", "--format=%B", ref)
	if err != nil {
		return task.GitCommit{}, err
	}
	hash = strings.TrimSpace(hash)
	message = strings.TrimRight(message, "\n")
	var commitType string
	if match := conventionalType.FindStringSubmatch(message); match != nil {
		commitType = match[1]
	}
	return task.GitCommit{Type: commitType, Message: message, Hash: hash}, nil
}

func (g *Client) CommitBookkeeping(ctx context.Context, path, message string) error {
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

func (g *Client) hasStagedChanges(ctx context.Context) (bool, error) {
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

func (g *Client) run(ctx context.Context, dir string, args ...string) (string, error) {
	//nolint:gosec // taskman invokes Git with its own validated operation arguments.
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
