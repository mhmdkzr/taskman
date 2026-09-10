package verify

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/urfave/cli/v3"

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
	full := append([]string{"taskman", "verified"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)
	out, err := runCmd(t, dir, "abc", "--check", "vet=ok", "--check", "test=ok")
	if err != nil {
		t.Fatalf("verify: %v\noutput:\n%s", err, out)
	}
	got, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if len(got.Verifications) != 1 {
		t.Fatalf("verifications = %d, want 1", len(got.Verifications))
	}
	if !got.Verifications[0].Passed() {
		t.Fatalf("verification should have passed")
	}
}

func TestCommandBeforeImplementationDone(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false, false)
	if _, err := runCmd(t, dir, "abc", "--check", "vet=ok"); err == nil {
		t.Fatal("verify before implementation: want error, got nil")
	}
}

func TestCommandInvalidCheckValue(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)
	if _, err := runCmd(t, dir, "abc", "--check", "vet=maybe"); err == nil {
		t.Fatal("verify with invalid check value: want error, got nil")
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir); err == nil {
		t.Fatal("verify without an id: want error, got nil")
	}
}

func TestCommandMissingCheck(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)
	if _, err := runCmd(t, dir, "abc"); err == nil {
		t.Fatal("verify without --check: want error, got nil")
	}
}
