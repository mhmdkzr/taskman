package merge

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
	full := append([]string{"taskman", "merge"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommand(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	out, err := runCmd(t, dir, "abc")
	if err != nil {
		t.Fatalf("merge: %v\noutput:\n%s", err, out)
	}
	got, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.State != task.StateCompleted {
		t.Fatalf("state = %v, want completed", got.State)
	}
}

func TestCommandWithCommitOverride(t *testing.T) {
	dir := newTestTaskDir(t, "abc", true)
	newCommit := "xyz789"
	out, err := runCmd(t, dir, "abc", "--commit", newCommit)
	if err != nil {
		t.Fatalf("merge: %v\noutput:\n%s", err, out)
	}
	got, err := store.Read(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.Git.Commit == nil {
		t.Fatalf("git.commit is nil")
	}
	if got.Git.Commit.Hash != newCommit {
		t.Fatalf("commit hash = %v, want %v", got.Git.Commit.Hash, newCommit)
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCmd(t, dir); err == nil {
		t.Fatal("merge without an id: want error, got nil")
	}
}

func TestCommandBeforeReviewDone(t *testing.T) {
	dir := newTestTaskDir(t, "abc", false)
	if _, err := runCmd(t, dir, "abc"); err == nil {
		t.Fatal("merge before review done: want error, got nil")
	}
}
