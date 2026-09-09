package abandon

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
)

func TestMain(m *testing.M) {
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

func runCmd(t *testing.T, gitDir string, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := &cli.Command{
		Name: "taskman",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "git-dir", Value: gitDir},
			&cli.StringFlag{Name: "tasks-dir", Value: gitDir},
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
	full := append([]string{"taskman", "abandon"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateStarted)
	out, err := runCmd(t, dir, "abc", "--reason", "test abandon")
	if err != nil {
		t.Fatalf("abandon: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.State != task.StateFailed {
		t.Fatalf("state = %v, want %v", got.State, task.StateFailed)
	}
	if got.FailureReason != "test abandon" {
		t.Fatalf("failure_reason = %q, want %q", got.FailureReason, "test abandon")
	}
}

func TestCommandMissingReason(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateStarted)
	if _, err := runCmd(t, dir, "abc"); err == nil {
		t.Fatal("abandon without --reason: want error, got nil")
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir); err == nil {
		t.Fatal("abandon without an id: want error, got nil")
	}
}
