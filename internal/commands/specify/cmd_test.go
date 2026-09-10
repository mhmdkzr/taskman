package specify

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
	full := append([]string{"taskman", "specify"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	out, err := runCmd(t, dir, "abc", "--result", "spec", "--done-when", "criteria")
	if err != nil {
		t.Fatalf("specify: %v\noutput:\n%s", err, out)
	}
	got, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.State != task.StateImplement {
		t.Fatalf("state = %v, want implement", got.State)
	}
	if got.Specification != "spec" {
		t.Fatalf("specification = %q, want spec", got.Specification)
	}
	if got.DoneWhen != "criteria" {
		t.Fatalf("done_when = %q, want criteria", got.DoneWhen)
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir, "--result", "spec", "--done-when", "criteria"); err == nil {
		t.Fatal("specify without an id: want error, got nil")
	}
}

func TestCommandMissingDoneWhen(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	if _, err := runCmd(t, dir, "abc", "--result", "spec"); err == nil {
		t.Fatal("specify without --done-when: want error, got nil")
	}
}

func TestCommandAlreadySpecified(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	if _, err := runCmd(t, dir, "abc", "--result", "spec", "--done-when", "criteria"); err == nil {
		t.Fatal("specify already-specified task: want error, got nil")
	}
}
