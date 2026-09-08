package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
)

func TestMain(m *testing.M) {
	// cli.Command.Run calls cli.HandleExitCoder on any returned error that
	// implements cli.ExitCoder, which by default calls os.Exit - fine for
	// the real binary, fatal for a test binary. The library's own Exit doc
	// names this exact override as the intended way to test it.
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

// newTestRepo creates a fresh git repository with one commit, and returns
// its path.
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	// .worktrees/ is where task create checks out each task's worktree
	// (§5) - it must be gitignored, or every task after the first makes
	// the clean-working-tree precondition fail on the untracked directory
	// the previous one left behind.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".worktrees/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	git("add", "README.md", ".gitignore")
	git("commit", "-q", "-m", "chore: init")
	return dir
}

// run invokes the taskman CLI in-process against gitDir, returning stdout.
// It fails the test if the command errors.
func runTaskman(t *testing.T, gitDir string, args ...string) string {
	t.Helper()
	out, err := runTaskmanErr(gitDir, args...)
	if err != nil {
		t.Fatalf("taskman %v: %v\noutput:\n%s", args, err, out)
	}
	return out
}

// runErr is like run but returns the error instead of failing the test.
// Every directory flag is pinned under gitDir explicitly, rather than
// relying on the test process's current working directory matching it.
func runTaskmanErr(gitDir string, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := rootCommand()
	fixStdout(cmd, &buf)
	full := append([]string{
		"taskman",
		"--git-dir", gitDir,
		"--tasks-dir", filepath.Join(gitDir, ".tasks"),
		"--worktrees-dir", filepath.Join(gitDir, ".worktrees"),
	}, args...)
	err := cmd.Run(context.Background(), full)
	return buf.String(), err
}

// fixStdout redirects every command's output through w, recursively.
func fixStdout(cmd *cli.Command, w *bytes.Buffer) {
	cmd.Writer = w
	cmd.ErrWriter = w
	for _, sub := range cmd.Commands {
		fixStdout(sub, w)
	}
}

func TestCLIFullLifecycle(t *testing.T) {
	dir := newTestRepo(t)

	created := runTaskman(t, dir, "task", "create", "--definition", "Fix doc drift", "--title", "Fix Doc Drift")
	if !strings.Contains(created, "Created task") {
		t.Fatalf("create output = %q", created)
	}

	id := onlyTaskID(t, dir)

	runTaskman(t, dir, "task", "specify", id, "--result", "Update the README", "--done-when", "README reflects reality")
	runTaskman(t, dir, "task", "implement", id)
	runTaskman(t, dir, "task", "verify", id, "--check", "vet=ok")
	runTaskman(t, dir, "task", "review", "record", id, "--approved=true")

	worktree := filepath.Join(dir, ".worktrees", id)
	gitCommit(t, worktree, "docs: update readme")

	runTaskman(t, dir, "task", "commit", id)
	runTaskman(t, dir, "task", "review", "approve", id, "--comment", "LGTM")

	mergeInto(t, dir, "task/"+id)
	runTaskman(t, dir, "task", "merge", id)

	final := getTaskJSON(t, dir, id)
	if final.State != task.StateCompleted {
		t.Fatalf("final state = %v, want completed", final.State)
	}
	if final.Status.Merge.State != task.StageDone {
		t.Fatalf("merge status = %+v, want done", final.Status.Merge)
	}

	next := runTaskman(t, dir, "task", "next", id)
	if !strings.Contains(next, "complete") {
		t.Fatalf("next output = %q, want it to say the task is complete", next)
	}
}

// TestCLIFullLifecycleTrunk mirrors TestCLIFullLifecycle, but with --trunk:
// the task works directly on gitDir/the current branch instead of an
// isolated worktree, so there's no separate branch to git-merge back in.
func TestCLIFullLifecycleTrunk(t *testing.T) {
	dir := newTestRepo(t)

	created := runTaskman(t, dir, "task", "create",
		"--definition", "Fix doc drift", "--title", "Fix Doc Drift", "--trunk")
	if !strings.Contains(created, "Created task") {
		t.Fatalf("create output = %q", created)
	}

	id := onlyTaskID(t, dir)

	before := getTaskJSON(t, dir, id)
	if before.Git.Worktree != dir {
		t.Fatalf("worktree = %q, want %q (the repo root)", before.Git.Worktree, dir)
	}
	if before.Git.Branch == "" {
		t.Fatalf("branch is empty")
	}
	if _, err := os.Stat(filepath.Join(dir, ".worktrees")); err == nil {
		t.Fatalf("--worktrees-dir was created despite --trunk")
	}

	runTaskman(t, dir, "task", "specify", id, "--result", "Update the README", "--done-when", "README reflects reality")
	runTaskman(t, dir, "task", "implement", id)
	runTaskman(t, dir, "task", "verify", id, "--check", "vet=ok")
	runTaskman(t, dir, "task", "review", "record", id, "--approved=true")

	gitCommit(t, dir, "docs: update readme")

	runTaskman(t, dir, "task", "commit", id)
	runTaskman(t, dir, "task", "review", "approve", id, "--comment", "LGTM")
	runTaskman(t, dir, "task", "merge", id)

	final := getTaskJSON(t, dir, id)
	if final.State != task.StateCompleted {
		t.Fatalf("final state = %v, want completed", final.State)
	}
	if final.Status.Merge.State != task.StageDone {
		t.Fatalf("merge status = %+v, want done", final.Status.Merge)
	}
}

func TestCLIDirtyWorkingTreeRefused(t *testing.T) {
	dir := newTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := runTaskmanErr(dir, "task", "create", "--definition", "x")
	if err == nil {
		t.Fatal("create on dirty tree: want error, got nil")
	}
}

func TestCLIInvalidTransitionExitsNonZero(t *testing.T) {
	dir := newTestRepo(t)
	runTaskman(t, dir, "task", "create", "--definition", "x")
	id := onlyTaskID(t, dir)

	_, err := runTaskmanErr(dir, "task", "implement", id)
	if err == nil {
		t.Fatal("implement before specify: want error, got nil")
	}
	if _, ok := errors.AsType[cli.ExitCoder](err); !ok {
		t.Fatalf("err = %v (%T), want an ExitCoder", err, err)
	}
}

func TestCLIList(t *testing.T) {
	dir := newTestRepo(t)
	runTaskman(t, dir, "task", "create", "--definition", "one", "--title", "One")
	commitTaskFiles(t, dir)
	runTaskman(t, dir, "task", "create", "--definition", "two", "--title", "Two", "--label", "priority=high")
	commitTaskFiles(t, dir)

	out := runTaskman(t, dir, "task", "list", "--json")
	var tasks []task.Task
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	if len(tasks) != 2 {
		t.Fatalf("list len = %d, want 2", len(tasks))
	}

	filtered := runTaskman(t, dir, "task", "list", "--label", "priority=high", "--json")
	var filteredTasks []task.Task
	if err := json.Unmarshal([]byte(filtered), &filteredTasks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(filteredTasks) != 1 || filteredTasks[0].Title != "Two" {
		t.Fatalf("filtered = %+v, want just Two", filteredTasks)
	}
}

func onlyTaskID(t *testing.T, gitDir string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(gitDir, ".tasks"))
	if err != nil {
		t.Fatalf("read tasks dir: %v", err)
	}
	var ids []string
	for _, e := range entries {
		if before, ok := strings.CutSuffix(e.Name(), ".yaml"); ok {
			ids = append(ids, before)
		}
	}
	if len(ids) != 1 {
		t.Fatalf("tasks dir has %d task files, want exactly 1: %v", len(ids), ids)
	}
	return ids[0]
}

func commitTaskFiles(t *testing.T, gitDir string) {
	t.Helper()
	for _, args := range [][]string{
		{"add", ".tasks"},
		{"commit", "-q", "-m", "chore: track task"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = gitDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func gitCommit(t *testing.T, worktree, message string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = worktree
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(worktree, "CHANGE.md"), []byte("change\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "CHANGE.md")
	run("commit", "-q", "-m", message)
}

func mergeInto(t *testing.T, gitDir, branch string) {
	t.Helper()
	cmd := exec.Command("git", "merge", "--no-ff", branch, "-m", "merge "+branch)
	cmd.Dir = gitDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git merge: %v: %s", err, out)
	}
}

func getTaskJSON(t *testing.T, gitDir, id string) task.Task {
	t.Helper()
	out := runTaskman(t, gitDir, "task", "get", id, "--json")
	var tk task.Task
	if err := json.Unmarshal([]byte(out), &tk); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	return tk
}
