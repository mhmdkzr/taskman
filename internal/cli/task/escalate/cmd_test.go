package escalate

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
	full := append([]string{"taskman", "escalate"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateStarted)
	out, err := runCmd(t, dir, "abc", "--stage", "implementation", "--reason", "agent gave up")
	if err != nil {
		t.Fatalf("escalate: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.State != task.StateBlocked {
		t.Fatalf("state = %v, want blocked", got.State)
	}
	if got.Blocked == nil {
		t.Fatal("Blocked is nil, want non-nil")
	}
	if got.Blocked.Stage != "implementation" {
		t.Fatalf("blocked.stage = %v, want implementation", got.Blocked.Stage)
	}
	if got.Blocked.Reason != "agent gave up" {
		t.Fatalf("blocked.reason = %v, want 'agent gave up'", got.Blocked.Reason)
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir, "--stage", "implementation", "--reason", "agent gave up"); err == nil {
		t.Fatal("escalate without an id: want error, got nil")
	}
}

func TestCommandMissingReason(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateStarted)
	if _, err := runCmd(t, dir, "abc", "--stage", "implementation"); err == nil {
		t.Fatal("escalate without --reason: want error, got nil")
	}
}

func TestCommandAlreadyTerminal(t *testing.T) {
	dir := newTestTaskDir(t, "abc", task.StateFailed)
	if _, err := runCmd(t, dir, "abc", "--stage", "implementation", "--reason", "agent gave up"); err == nil {
		t.Fatal("escalate failed task: want error, got nil")
	}
}
