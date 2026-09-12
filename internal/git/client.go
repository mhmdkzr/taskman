// Package git provides taskman's imperative Git adapter.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Client wraps the Git operations taskman performs itself.
type Client struct {
	dir string
}

func NewClient(dir string) *Client {
	return &Client{dir: dir}
}

// ReadCommit reads ref's hash and message from worktreeDir. An empty ref
// means HEAD. At is set to the time of the call, not the commit's own
// author/commit date - taskman records when it observed the commit, not
// when it was made.
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
	return task.GitCommit{
		Hash:    strings.TrimSpace(hash),
		Message: strings.TrimRight(message, "\n"),
		At:      time.Now().UTC(),
	}, nil
}

// ReadBranchCommit reads ref's hash and message from the client's
// repository root - for refs that live in the main repository (like the
// target branch of a merge) rather than in one of a task's implementation
// worktrees.
func (g *Client) ReadBranchCommit(ctx context.Context, ref string) (task.GitCommit, error) {
	return g.ReadCommit(ctx, g.dir, ref)
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
