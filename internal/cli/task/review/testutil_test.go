package review

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/task"
)

func TestMain(m *testing.M) {
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

// newTestRepo creates a fresh git repository with one commit, and a task
// "abc" that's already through implementation - ready for review record.
// Setup goes through internal/task directly (not internal/cli/task's CLI
// commands) so this package never has to import the package that imports
// it.
func newTestRepo(t *testing.T) (dir, id string) {
	t.Helper()
	dir = t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	git("add", "README.md")
	git("commit", "-q", "-m", "chore: init")

	repo := task.NewRepo(filepath.Join(dir, ".tasks"))
	gitClient := task.NewGit(dir)
	tk, err := task.Create(context.Background(), repo, gitClient, filepath.Join(dir, ".worktrees"), task.CreateRequest{
		Definition: "test task",
		Title:      "Test Task",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := task.Specify(repo, tk.ID, task.SpecifyRequest{Result: "spec", DoneWhen: "criteria"}); err != nil {
		t.Fatalf("specify: %v", err)
	}
	if _, err := task.Implement(repo, tk.ID); err != nil {
		t.Fatalf("implement: %v", err)
	}
	return dir, tk.ID
}

// runCmd wraps cmd in a minimal root command carrying the global flags
// every taskman command relies on, and runs it against gitDir.
func runCmd(t *testing.T, gitDir string, cmd *cli.Command, args ...string) error {
	t.Helper()
	var buf bytes.Buffer
	root := &cli.Command{
		Name: "taskman",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "git-dir", Value: "."},
			&cli.StringFlag{Name: "tasks-dir", Value: ".tasks"},
			&cli.StringFlag{Name: "worktrees-dir", Value: ".worktrees"},
			&cli.BoolFlag{Name: "json"},
		},
		Commands: []*cli.Command{cmd},
	}
	var setWriters func(c *cli.Command)
	setWriters = func(c *cli.Command) {
		c.Writer = &buf
		c.ErrWriter = &buf
		for _, sub := range c.Commands {
			setWriters(sub)
		}
	}
	setWriters(root)

	full := append([]string{
		"taskman",
		"--git-dir", gitDir,
		"--tasks-dir", filepath.Join(gitDir, ".tasks"),
		"--worktrees-dir", filepath.Join(gitDir, ".worktrees"),
		cmd.Name,
	}, args...)
	return root.Run(context.Background(), full)
}

func getTask(t *testing.T, gitDir, id string) task.Task {
	t.Helper()
	tk, err := task.NewRepo(filepath.Join(gitDir, ".tasks")).Get(id)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	return tk
}
