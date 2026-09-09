package commit

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
)

func TestMain(m *testing.M) {
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

func runCmd(t *testing.T, gitDir, tasksDir string, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := &cli.Command{
		Name: "taskman",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "git-dir", Value: gitDir},
			&cli.StringFlag{Name: "tasks-dir", Value: tasksDir},
			&cli.BoolFlag{Name: "json"},
		},
		Commands: []*cli.Command{Command()},
	}
	root.Writer = &buf
	root.ErrWriter = &buf
	for _, sub := range root.Commands {
		sub.Writer = &buf
		sub.ErrWriter = &buf
	}
	full := append([]string{"taskman", "commit"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	gitDir := newTestRepo(t)
	gitClient := task.NewGit(gitDir)
	ctx := context.Background()

	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	worktree, branch, err := gitClient.CreateWorktree(ctx, worktreesDir, "abc", "abc")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	tasksDir := t.TempDir()
	newTestTaskDir(t, tasksDir, worktree, branch)

	// Make a real commit in the worktree
	if err := os.WriteFile(filepath.Join(worktree, "FILE.md"), []byte("content\n"), 0o644); err != nil {
		t.Fatalf("write FILE.md: %v", err)
	}
	commitInWorktree := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = worktree
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	commitInWorktree("add", "FILE.md")
	commitInWorktree("commit", "-q", "-m", "fix: correct behavior")

	out, err := runCmd(t, gitDir, tasksDir, "abc")
	if err != nil {
		t.Fatalf("commit: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(tasksDir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.Git.Commit == nil {
		t.Fatal("git.commit is nil")
	}
	if got.Git.Commit.Type != "fix" {
		t.Errorf("commit.type = %q, want fix", got.Git.Commit.Type)
	}
	if got.Status.Review.State != task.StagePending {
		t.Errorf("review.state = %v, want pending", got.Status.Review.State)
	}
}

func TestCommandMissingID(t *testing.T) {
	gitDir := newTestRepo(t)
	tasksDir := t.TempDir()
	if _, err := runCmd(t, gitDir, tasksDir); err == nil {
		t.Fatal("commit without an id: want error, got nil")
	}
}

func TestCommandRequiresVerificationDone(t *testing.T) {
	gitDir := newTestRepo(t)
	gitClient := task.NewGit(gitDir)
	ctx := context.Background()

	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	worktree, branch, err := gitClient.CreateWorktree(ctx, worktreesDir, "abc", "abc")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	tasksDir := t.TempDir()
	tk := task.Task{
		ID:         "abc",
		State:      task.StateStarted,
		Definition: "def",
		Status: task.Status{
			Definition:   task.StageStatus{State: task.StageDone},
			Verification: task.StageStatus{State: task.StagePending},
		},
		Git: task.Git{
			Worktree: worktree,
			Branch:   branch,
		},
	}
	if err := task.WriteTaskFile(tasksDir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	if _, err := runCmd(t, gitDir, tasksDir, "abc"); err == nil {
		t.Error("commit before verification.state == done: want error, got nil")
	}
}
