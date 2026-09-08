package delete

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/task"
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
	full := append([]string{"taskman", "delete"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommandDeletesExistingTask(t *testing.T) {
	dir := t.TempDir()
	tk := task.Task{
		ID:         "abc",
		State:      task.StateStarted,
		Definition: "test task",
		Status: task.Status{
			Definition: task.StageStatus{State: task.StageDone},
		},
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	out, err := runCmd(t, dir, "abc")
	if err != nil {
		t.Fatalf("delete: %v\noutput:\n%s", err, out)
	}

	// Verify output message
	if out != "deleted task abc\n" {
		t.Fatalf("output = %q, want %q", out, "deleted task abc\n")
	}

	// Verify task is gone
	if _, err := task.ReadTask(dir, "abc"); err == nil {
		t.Fatal("read task after delete: want error, got nil")
	}
}

func TestCommandDeleteNonexistent(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, dir, "nonexistent")
	if err == nil {
		t.Fatal("delete nonexistent: want error, got nil")
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir); err == nil {
		t.Fatal("delete without an id: want error, got nil")
	}
}
