package commit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
)

// newTestRepo creates a fresh git repository with one commit and a
// .gitignore excluding .worktrees/ and MutateTask's *.lock files, and
// returns its path.
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
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".worktrees/\n*.lock\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	git("add", "README.md", ".gitignore")
	git("commit", "-q", "-m", "chore: init")
	return dir
}

func newTestTaskDir(t *testing.T, tasksDir, worktree, branch string) {
	t.Helper()
	tk := task.Task{
		ID:         "abc",
		State:      task.StateStarted,
		Definition: "def",
		Status: task.Status{
			Definition:   task.StageStatus{State: task.StageDone},
			Verification: task.StageStatus{State: task.StageDone},
		},
		Git: task.Git{
			Worktree: worktree,
			Branch:   branch,
		},
	}
	if err := task.WriteTaskFile(tasksDir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
}

func TestCommit(t *testing.T) {
	gitDir := newTestRepo(t)
	gitClient := task.NewGit(gitDir)
	ctx := context.Background()

	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	worktree, branch, err := gitClient.CreateWorktree(ctx, worktreesDir, "abc")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	tasksDir := t.TempDir()
	newTestTaskDir(t, tasksDir, worktree, branch)

	// Make a real commit in the worktree
	if err := os.WriteFile(filepath.Join(worktree, "FIX.md"), []byte("fix\n"), 0o644); err != nil {
		t.Fatalf("write FIX.md: %v", err)
	}
	commitInWorktree := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = worktree
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	commitInWorktree("add", "FIX.md")
	commitInWorktree("commit", "-q", "-m", "feat: add fix")

	got, err := Commit(ctx, tasksDir, gitClient, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if got.Git.Commit == nil {
		t.Fatal("git.commit is nil")
	}
	if got.Git.Commit.Type != "feat" {
		t.Errorf("commit.type = %q, want feat", got.Git.Commit.Type)
	}
	if got.Git.Commit.Hash == "" {
		t.Error("commit.hash is empty")
	}
	if got.Status.Review.State != task.StagePending {
		t.Errorf("review.state = %v, want pending", got.Status.Review.State)
	}
}

func TestCommitAutoApproveSkipsHumanReview(t *testing.T) {
	gitDir := newTestRepo(t)
	gitClient := task.NewGit(gitDir)
	ctx := context.Background()

	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	worktree, branch, err := gitClient.CreateWorktree(ctx, worktreesDir, "abc")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	tasksDir := t.TempDir()
	tk := task.Task{
		ID:          "abc",
		State:       task.StateStarted,
		Definition:  "def",
		AutoApprove: true,
		Status: task.Status{
			Definition:   task.StageStatus{State: task.StageDone},
			Verification: task.StageStatus{State: task.StageDone},
		},
		Git: task.Git{Worktree: worktree, Branch: branch},
	}
	if err := task.WriteTaskFile(tasksDir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(worktree, "FIX.md"), []byte("fix\n"), 0o644); err != nil {
		t.Fatalf("write FIX.md: %v", err)
	}
	commitInWorktree := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = worktree
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	commitInWorktree("add", "FIX.md")
	commitInWorktree("commit", "-q", "-m", "feat: add fix")

	got, err := Commit(ctx, tasksDir, gitClient, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if got.Status.Review.State != task.StageDone {
		t.Errorf(
			"review.state = %v, want done (auto-approve skips the pending human-review wait)",
			got.Status.Review.State,
		)
	}
	if len(got.HumanReviews) != 0 {
		t.Errorf("human_reviews = %+v, want empty - no human reviewed this", got.HumanReviews)
	}
}

func TestCommitAutoApproveTrunkCompletesTask(t *testing.T) {
	gitDir := newTestRepo(t)
	gitClient := task.NewGit(gitDir)
	ctx := context.Background()

	tasksDir := gitDir
	tk := task.Task{
		ID:          "abc",
		State:       task.StateStarted,
		Definition:  "def",
		AutoApprove: true,
		Status: task.Status{
			Definition:   task.StageStatus{State: task.StageDone},
			Verification: task.StageStatus{State: task.StageDone},
		},
		Git: task.Git{Worktree: gitDir, Branch: "main", Trunk: true},
	}
	if err := task.WriteTaskFile(tasksDir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(gitDir, "FIX.md"), []byte("fix\n"), 0o644); err != nil {
		t.Fatalf("write FIX.md: %v", err)
	}
	commitInRepo := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = gitDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	commitInRepo("add", "FIX.md")
	commitInRepo("commit", "-q", "-m", "feat: add fix")

	got, err := Commit(ctx, tasksDir, gitClient, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if got.Status.Review.State != task.StageDone {
		t.Errorf("review.state = %v, want done", got.Status.Review.State)
	}
	if got.Status.Merge.State != task.StageDone {
		t.Errorf("merge.state = %v, want done - a trunk task has nothing left to merge", got.Status.Merge.State)
	}
	if got.State != task.StateCompleted {
		t.Errorf("state = %v, want completed", got.State)
	}
	clean, err := gitClient.IsClean(ctx)
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean {
		t.Fatal("working tree not clean after Commit completed the task: want the task file committed")
	}
}

func TestCommitRequiresVerificationDone(t *testing.T) {
	gitDir := newTestRepo(t)
	gitClient := task.NewGit(gitDir)
	ctx := context.Background()

	worktreesDir := filepath.Join(t.TempDir(), "worktrees")
	worktree, branch, err := gitClient.CreateWorktree(ctx, worktreesDir, "abc")
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	tasksDir := t.TempDir()
	tk := task.Task{
		ID:         "abc",
		State:      task.StateStarted,
		Definition: "def",
		Status: task.Status{
			Definition:   task.StageStatus{State: task.StageDone},
			Verification: task.StageStatus{State: task.StagePending},
		},
		Git: task.Git{
			Worktree: worktree,
			Branch:   branch,
		},
	}
	if err := task.WriteTaskFile(tasksDir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	if _, err := Commit(ctx, tasksDir, gitClient, Request{ID: "abc"}); err == nil {
		t.Error("Commit before verification.state == done: want error, got nil")
	}
}
