package record

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
	full := append([]string{"taskman", "record"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)
	out, err := runCmd(t, dir, "abc", "--approved")
	if err != nil {
		t.Fatalf("record: %v\noutput:\n%s", err, out)
	}
	got, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.State != task.StateCommit {
		t.Fatalf("state = %v, want commit", got.State)
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir); err == nil {
		t.Fatal("record without an id: want error, got nil")
	}
}

func TestCommandMissingApproved(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)
	if _, err := runCmd(t, dir, "abc"); err == nil {
		t.Fatal("record without --approved: want error, got nil")
	}
}

func TestCommandTwoRejectionsBlock(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true, false)

	// First rejection
	out1, err := runCmd(t, dir, "abc", "--approved=false", "--finding", "main.go=unused var")
	if err != nil {
		t.Fatalf("first record: %v\noutput:\n%s", err, out1)
	}

	got1, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read task after first rejection: %v", err)
	}
	if got1.State == task.StateBlocked {
		t.Fatal("first rejection should not block task")
	}
	got1.State = task.StateAutomatedReview
	if err := store.Write(dir, got1); err != nil {
		t.Fatalf("prepare second review: %v", err)
	}

	// Second rejection
	out2, err := runCmd(t, dir, "abc", "--approved=false", "--finding", "main.go=still unused")
	if err != nil {
		t.Fatalf("second record: %v\noutput:\n%s", err, out2)
	}

	got2, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read task after second rejection: %v", err)
	}
	if got2.State != task.StateBlocked {
		t.Fatalf("state = %v, want blocked", got2.State)
	}
	if got2.Blocked == nil {
		t.Fatal("blocked should not be nil")
	}
	if got2.Blocked.Stage != task.StageVerification {
		t.Fatalf("blocked.stage = %v, want verification", got2.Blocked.Stage)
	}
}
