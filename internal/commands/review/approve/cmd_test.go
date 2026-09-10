package approve

import (
	"bytes"
	"context"
	"os"
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
	full := append([]string{"taskman", "approve"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc")
	if err != nil {
		t.Fatalf("approve: %v\noutput:\n%s", err, out)
	}
	got, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.State != task.StateMerge {
		t.Fatalf("state = %v, want merge", got.State)
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir); err == nil {
		t.Fatal("approve without an id: want error, got nil")
	}
}

func TestCommandAlreadyApproved(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	if _, err := runCmd(t, dir, "abc"); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	if _, err := runCmd(t, dir, "abc"); err == nil {
		t.Fatal("second approve on already-approved review: want error, got nil")
	}
}
