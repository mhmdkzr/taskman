package reject

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
	full := append([]string{"taskman", "reject"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	out, err := runCmd(t, dir, "abc", "--reason", "needs more work")
	if err != nil {
		t.Fatalf("reject: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.Status.Review.State != task.StageInProgress {
		t.Fatalf("review.state = %v, want in_progress", got.Status.Review.State)
	}
}

func TestCommandMissingReason(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	if _, err := runCmd(t, dir, "abc"); err == nil {
		t.Fatal("reject without --reason: want error, got nil")
	}
}

func TestCommandNotPending(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	if _, err := runCmd(t, dir, "abc", "--reason", "needs more work"); err == nil {
		t.Fatal("reject when not pending: want error, got nil")
	}
}
