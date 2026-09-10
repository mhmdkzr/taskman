package next

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
	full := append([]string{"taskman", "next"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := t.TempDir()
	if err := store.Write(
		dir,
		task.Task{ID: "abc", State: task.StateSpecify, Definition: "definition"},
	); err != nil {
		t.Fatalf("write task: %v", err)
	}

	out, err := runCmd(t, dir, "abc")
	if err != nil {
		t.Fatalf("next: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "specification") {
		t.Fatalf("next output = %q, want it to mention drafting a specification", out)
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir); err == nil {
		t.Fatal("next without an id: want error, got nil")
	}
}

func TestCommandUnknownTask(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir, "does-not-exist"); err == nil {
		t.Fatal("next on unknown task: want error, got nil")
	}
}
