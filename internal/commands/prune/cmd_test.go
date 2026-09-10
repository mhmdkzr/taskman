package prune

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func TestMain(m *testing.M) {
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

func runCmd(t *testing.T, tasksDir string, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := &cli.Command{
		Name: "taskman",
		Flags: []cli.Flag{
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
	full := append([]string{"taskman", "prune"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommandPruneDeletesCompleted(t *testing.T) {
	dir := t.TempDir()
	if err := store.Write(dir, task.Task{ID: "done", State: task.StateCompleted, Definition: "def"}); err != nil {
		t.Fatalf("write task: %v", err)
	}
	if err := store.Write(
		dir,
		task.Task{ID: "started", State: task.StateImplement, Definition: "def"},
	); err != nil {
		t.Fatalf("write task: %v", err)
	}

	out, err := runCmd(t, dir)
	if err != nil {
		t.Fatalf("prune: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "deleted done\n") {
		t.Fatalf("output = %q, want it to contain %q", out, "deleted done\n")
	}
	if _, err := store.Read(dir, "done"); err == nil {
		t.Fatal("read done after prune: want error, got nil")
	}
	if _, err := store.Read(dir, "started"); err != nil {
		t.Fatalf("read started after prune: want present, got %v", err)
	}
}

func TestCommandPruneDryRun(t *testing.T) {
	dir := t.TempDir()
	if err := store.Write(dir, task.Task{ID: "done", State: task.StateCompleted, Definition: "def"}); err != nil {
		t.Fatalf("write task: %v", err)
	}

	out, err := runCmd(t, dir, "--dry-run")
	if err != nil {
		t.Fatalf("prune --dry-run: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "would delete done\n") {
		t.Fatalf("output = %q, want it to contain %q", out, "would delete done\n")
	}
	if _, err := store.Read(dir, "done"); err != nil {
		t.Fatalf("read done after dry-run: want present, got %v", err)
	}
}

func TestCommandPruneNone(t *testing.T) {
	dir := t.TempDir()
	out, err := runCmd(t, dir)
	if err != nil {
		t.Fatalf("prune: %v\noutput:\n%s", err, out)
	}
	if out != "no completed tasks\n" {
		t.Fatalf("output = %q, want %q", out, "no completed tasks\n")
	}
}
