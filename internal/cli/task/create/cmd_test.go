package create

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestMain(m *testing.M) {
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

func runCmd(t *testing.T, gitDir string, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	tasksDir := filepath.Join(gitDir, ".tasks")
	worktreesDir := filepath.Join(gitDir, ".worktrees")

	root := &cli.Command{
		Name: "taskman",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "git-dir", Value: gitDir},
			&cli.StringFlag{Name: "tasks-dir", Value: tasksDir},
			&cli.StringFlag{Name: "worktrees-dir", Value: worktreesDir},
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
	full := append([]string{"taskman", "create"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommandCreate(t *testing.T) {
	gitDir := newTestRepo(t)
	tasksDir := filepath.Join(gitDir, ".tasks")

	out, err := runCmd(t, gitDir, "--definition", "test task", "--title", "Test Task")
	if err != nil {
		t.Fatalf("create: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Created task") {
		t.Fatalf("output missing 'Created task', got:\n%s", out)
	}
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("read tasks dir: %v", err)
		}
	} else if len(entries) != 1 {
		t.Fatalf("tasks dir has %d entries, want 1", len(entries))
	}
}

func TestCommandMissingDefinition(t *testing.T) {
	gitDir := newTestRepo(t)

	_, err := runCmd(t, gitDir, "--title", "Test Task")
	if err == nil {
		t.Fatal("create without definition: want error, got nil")
	}
}

func TestCommandDirtyWorkingTree(t *testing.T) {
	gitDir := newTestRepo(t)

	// Write an uncommitted file to make working tree dirty
	if err := os.WriteFile(filepath.Join(gitDir, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	_, err := runCmd(t, gitDir, "--definition", "test task", "--title", "Test Task")
	if err == nil {
		t.Fatal("create with dirty working tree: want error, got nil")
	}
}
