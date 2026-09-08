package task

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestMain(m *testing.M) {
	// cli.Command.Run calls cli.HandleExitCoder on any error implementing
	// cli.ExitCoder, which by default calls os.Exit - fine for the real
	// binary, fatal for a test binary.
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

// newTestRepo creates a fresh git repository with one commit and a
// .gitignore excluding .worktrees/ (required before any task create -
// see internal/task's design notes), and returns its path.
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
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
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".worktrees/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	git("add", "README.md", ".gitignore")
	git("commit", "-q", "-m", "chore: init")
	return dir
}

// runCmd wraps cmd in a minimal root command carrying the global flags
// every taskman command relies on, and runs it against gitDir. args are
// the arguments after the command's own name (flags, positional id, ...).
func runCmd(t *testing.T, gitDir string, cmd *cli.Command, args ...string) (string, error) {
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
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

// onlyTaskID returns the id of the single task file under gitDir/.tasks.
func onlyTaskID(t *testing.T, gitDir string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(gitDir, ".tasks"))
	if err != nil {
		t.Fatalf("read tasks dir: %v", err)
	}
	var ids []string
	for _, e := range entries {
		if before, ok := strings.CutSuffix(e.Name(), ".yaml"); ok {
			ids = append(ids, before)
		}
	}
	if len(ids) != 1 {
		t.Fatalf("tasks dir has %d task files, want exactly 1: %v", len(ids), ids)
	}
	return ids[0]
}

// createTask creates a task and returns its id.
func createTask(t *testing.T, gitDir string) string {
	t.Helper()
	if _, err := runCmd(t, gitDir, Create(), "--definition", "test task", "--title", "Test Task"); err != nil {
		t.Fatalf("create: %v", err)
	}
	return onlyTaskID(t, gitDir)
}

// readyForVerify advances id through specify and implement, so verify (or
// review record) preconditions are met.
func readyForVerify(t *testing.T, gitDir, id string) {
	t.Helper()
	if _, err := runCmd(t, gitDir, Specify(), id, "--result", "spec", "--done-when", "criteria"); err != nil {
		t.Fatalf("specify: %v", err)
	}
	if _, err := runCmd(t, gitDir, Implement(), id); err != nil {
		t.Fatalf("implement: %v", err)
	}
}
