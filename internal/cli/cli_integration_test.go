package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
	jsonview "github.com/mhmdkzr/taskman/internal/task/view/json"
	"github.com/mhmdkzr/taskman/internal/utils"
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
	git("add", "README.md")
	git("commit", "-q", "-m", "chore: init")
	return dir
}

// run invokes the taskman CLI in-process against gitDir/db, returning
// stdout. It fails the test if the command errors.
func runTaskman(t *testing.T, gitDir, dbPath string, args ...string) string {
	t.Helper()
	out, err := runTaskmanErr(gitDir, dbPath, args...)
	if err != nil {
		t.Fatalf("taskman %v: %v\noutput:\n%s", args, err, out)
	}
	return out
}

// runTaskmanErr is like run but returns the error instead of failing the
// test.
func runTaskmanErr(gitDir, dbPath string, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := rootCommand()
	fixStdout(cmd, &buf)
	full := append([]string{
		"taskman",
		"--git-dir", gitDir,
		"--db", dbPath,
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

// createWorktree creates a git worktree at worktreesDir/name on a new
// branch task/name, and returns its path.
func createWorktree(t *testing.T, gitDir, worktreesDir, name string) string {
	t.Helper()
	worktree := filepath.Join(worktreesDir, name)
	cmd := exec.Command("git", "worktree", "add", worktree, "-b", "task/"+name)
	cmd.Dir = gitDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v: %s", err, out)
	}
	return worktree
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

func getTaskJSON(t *testing.T, gitDir, dbPath, id string) task.Task {
	t.Helper()
	out := runTaskman(t, gitDir, dbPath, "get", "--id", id, "--json")
	var doc jsonview.Document
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	return doc.Task
}

func TestCLIFullLifecycle(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	worktreesDir := t.TempDir()

	created := runTaskman(t, dir, db, "create", "--description", "Fix doc drift", "--title", "Fix Doc Drift")
	if !strings.Contains(created, "specify") {
		t.Fatalf("create output = %q, want it to mention the initial state", created)
	}

	id := onlyTaskID(t, dir, db)

	if next := runTaskman(t, dir, db, "next", "--id", id); !strings.Contains(next, "specify") {
		t.Fatalf("next output = %q, want it to mention the specify state", next)
	}

	runTaskman(t, dir, db, "specified", "--id", id, "--plan", "Update the README",
		"--agent-review", "--human-review")
	next := runTaskman(t, dir, db, "next", "--id", id)
	if !strings.Contains(next, "specification review agent approved") {
		t.Fatalf("next output = %q, want it to list the agent-review command", next)
	}
	runTaskman(t, dir, db, "specification", "review", "agent", "approved", "--id", id, "--comment", "ok")
	runTaskman(t, dir, db, "specification", "review", "human", "approved", "--id", id, "--comment", "approved")

	worktree := createWorktree(t, dir, worktreesDir, id)
	runTaskman(t, dir, db, "implemented", "--id", id, "--worktree", worktree, "--branch", "task/"+id,
		"--unit", "--human-review")
	runTaskman(t, dir, db, "verified", "--id", id, "--unit", "ok")

	gitCommit(t, worktree, "docs: update readme")
	runTaskman(t, dir, db, "committed", "--id", id)
	runTaskman(t, dir, db, "implementation", "review", "human", "approved", "--id", id, "--comment", "lgtm")

	mergeInto(t, dir, "task/"+id)
	runTaskman(t, dir, db, "merged", "--id", id, "--target", "main")

	final := getTaskJSON(t, dir, db, id)
	if final.State() != task.StateCompleted {
		t.Fatalf("final state = %v, want completed", final.State())
	}

	if next := runTaskman(t, dir, db, "next", "--id", id); !strings.Contains(next, "complete") {
		t.Fatalf("next output = %q, want it to report the task as done", next)
	}
}

func TestCLIInvalidTransitionExitsNonZero(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	runTaskman(t, dir, db, "create", "--description", "x")
	id := onlyTaskID(t, dir, db)

	_, err := runTaskmanErr(dir, db, "implemented", "--id", id, "--worktree", "/x", "--branch", "b")
	if err == nil {
		t.Fatal("implement before specify: want error, got nil")
	}
}

func TestCLIImplementedRejectsReviewWithoutVerification(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	runTaskman(t, dir, db, "create", "--description", "x")
	id := onlyTaskID(t, dir, db)
	runTaskman(t, dir, db, "specified", "--id", id, "--plan", "p")

	_, err := runTaskmanErr(dir, db, "implemented", "--id", id, "--worktree", "/x", "--branch", "b", "--agent-review")
	if err == nil {
		t.Fatal("implemented with agent review but no verification: want error, got nil")
	}
}

func TestCLIMissingRequiredFlagExitsTwo(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")

	for _, args := range [][]string{
		{"get"},
		{"create", "--title", "no description"},
		{"merged", "--id", "01a094c6-313c-7bce-91b9-29287b30bf3e"},
		{"escalated", "--id", "01a094c6-313c-7bce-91b9-29287b30bf3e", "--stage", "s"},
		{"delete"},
	} {
		_, err := runTaskmanErr(dir, db, args...)
		if err == nil {
			t.Fatalf("taskman %v: error = nil, want a missing-required-flag error", args)
		}
		if got := utils.ExitCode(err); got != 2 {
			t.Fatalf("taskman %v: ExitCode = %d, want 2 (malformed input)", args, got)
		}
	}
}

func TestCLIList(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	runTaskman(t, dir, db, "create", "--description", "one", "--title", "One")
	runTaskman(t, dir, db, "create", "--description", "two", "--title", "Two")

	out := runTaskman(t, dir, db, "list", "--json")
	var docs []jsonview.Document
	if err := json.Unmarshal([]byte(out), &docs); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	if len(docs) != 2 {
		t.Fatalf("list = %+v, want 2 tasks", docs)
	}

	out = runTaskman(t, dir, db, "list")
	for _, want := range []string{" - One - [specify]", " - Two - [specify]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("list output missing %q\n---\n%s", want, out)
		}
	}
}

func TestCLIDelete(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	runTaskman(t, dir, db, "create", "--description", "x", "--title", "Title")
	id := onlyTaskID(t, dir, db)

	if out := runTaskman(t, dir, db, "delete", "--id", id); !strings.Contains(out, id) {
		t.Fatalf("delete output = %q, want it to mention the deleted task %s", out, id)
	}

	if _, err := runTaskmanErr(dir, db, "get", "--id", id); err == nil {
		t.Fatal("get after delete: want error, got nil")
	}

	out := runTaskman(t, dir, db, "list", "--json")
	var docs []jsonview.Document
	if err := json.Unmarshal([]byte(out), &docs); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	if len(docs) != 0 {
		t.Fatalf("list after delete = %+v, want no tasks", docs)
	}
}

func TestCLIPrune(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	id := completeTask(t, dir, db)

	// A dry run reports the completed task but leaves it in place.
	if out := runTaskman(t, dir, db, "prune", "--dry-run"); !strings.Contains(out, id) {
		t.Fatalf("prune --dry-run output = %q, want it to mention %s", out, id)
	}
	if _, err := runTaskmanErr(dir, db, "get", "--id", id); err != nil {
		t.Fatalf("get after prune --dry-run: %v, want the task to survive", err)
	}

	// A real prune removes it.
	if out := runTaskman(t, dir, db, "prune"); !strings.Contains(out, id) {
		t.Fatalf("prune output = %q, want it to mention %s", out, id)
	}
	if _, err := runTaskmanErr(dir, db, "get", "--id", id); err == nil {
		t.Fatal("get after prune: want error, got nil")
	}

	out := runTaskman(t, dir, db, "list", "--json")
	var docs []jsonview.Document
	if err := json.Unmarshal([]byte(out), &docs); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	if len(docs) != 0 {
		t.Fatalf("list after prune = %+v, want no tasks", docs)
	}
}

// completeTask creates a task and drives it to the completed state with no
// review gates, returning its id.
func completeTask(t *testing.T, dir, db string) string {
	t.Helper()
	runTaskman(t, dir, db, "create", "--description", "x", "--title", "Title")
	id := onlyTaskID(t, dir, db)

	runTaskman(t, dir, db, "specified", "--id", id, "--plan", "p")
	worktree := createWorktree(t, dir, t.TempDir(), id)
	runTaskman(t, dir, db, "implemented", "--id", id, "--worktree", worktree, "--branch", "task/"+id)
	gitCommit(t, worktree, "feat: work")
	runTaskman(t, dir, db, "committed", "--id", id)
	mergeInto(t, dir, "task/"+id)
	runTaskman(t, dir, db, "merged", "--id", id, "--target", "main")
	return id
}

func TestCLIGetMarkdown(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	runTaskman(t, dir, db, "create", "--description", "x", "--title", "Title")
	id := onlyTaskID(t, dir, db)

	out := runTaskman(t, dir, db, "get", "--id", id, "--md")
	for _, want := range []string{"# Task " + id, "**State:** specify", "**Title:** Title"} {
		if !strings.Contains(out, want) {
			t.Fatalf("get --md output missing %q\n---\n%s", want, out)
		}
	}
}

func TestCLIListMarkdown(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	runTaskman(t, dir, db, "create", "--description", "one", "--title", "One")
	runTaskman(t, dir, db, "create", "--description", "two", "--title", "Two")

	out := runTaskman(t, dir, db, "list", "--md")
	if got := strings.Count(out, "# Task "); got != 2 {
		t.Fatalf("list --md rendered %d task documents, want 2\n---\n%s", got, out)
	}
	if !strings.Contains(out, "\n---\n") {
		t.Fatalf("list --md output should separate task documents with a horizontal rule\n---\n%s", out)
	}
}

func TestCLIJSONAndMDConflict(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	runTaskman(t, dir, db, "create", "--description", "x")
	id := onlyTaskID(t, dir, db)

	_, err := runTaskmanErr(dir, db, "get", "--id", id, "--json", "--md")
	if err == nil {
		t.Fatal("--json with --md: want error, got nil")
	}
	if got := utils.ExitCode(err); got != 2 {
		t.Fatalf("--json with --md: ExitCode = %d, want 2 (malformed input)", got)
	}
}

func TestCLISkill(t *testing.T) {
	dir := newTestRepo(t)
	db := filepath.Join(dir, "tasks.db")
	out := runTaskman(t, dir, db, "skill")
	if !strings.Contains(out, "name: taskman") {
		t.Fatalf("skill output = %q, want it to contain the SKILL.md frontmatter", out)
	}
}

func onlyTaskID(t *testing.T, gitDir, dbPath string) string {
	t.Helper()
	out := runTaskman(t, gitDir, dbPath, "list", "--json")
	var docs []jsonview.Document
	if err := json.Unmarshal([]byte(out), &docs); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	if len(docs) != 1 {
		t.Fatalf("store has %d tasks, want exactly 1", len(docs))
	}
	return docs[0].Task.ID.String()
}
