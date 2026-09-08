package get

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
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
	full := append([]string{"taskman", "get"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "test-abc")
	out, err := runCmd(t, dir, "test-abc")
	if err != nil {
		t.Fatalf("get: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "test-abc") {
		t.Fatalf("output doesn't mention id: %s", out)
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir); err == nil {
		t.Fatal("get without an id: want error, got nil")
	}
}

func TestCommandUnknownID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir, "non-existent"); err == nil {
		t.Fatal("get unknown id: want error, got nil")
	}
}
