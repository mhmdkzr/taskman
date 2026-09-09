package next

import (
	"strings"
	"testing"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
)

// newTestTask writes a fresh task ready for the specify stage.
func newTestTask(t *testing.T, tasksDir, id string) {
	t.Helper()
	tk := task.Task{
		ID:         id,
		State:      task.StateCreated,
		Title:      "Test",
		Definition: "def",
		Status: task.Status{
			Definition:     task.StageStatus{State: task.StageDone},
			Specification:  task.StageStatus{State: task.StagePending},
			Implementation: task.StageStatus{State: task.StagePending},
			Verification:   task.StageStatus{State: task.StagePending},
			Review:         task.StageStatus{State: task.StagePending},
			Merge:          task.StageStatus{State: task.StagePending},
		},
		Git: task.Git{Worktree: "/tmp/wt", Branch: "task/" + id},
	}
	if err := task.WriteTaskFile(tasksDir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
}

// readyForVerify advances id through specify and implement directly via
// internal/task (not the CLI), so verify/review-record preconditions are
// met.
//
//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func readyForVerify(t *testing.T, tasksDir, id string) {
	t.Helper()
	if _, err := task.MutateTask(tasksDir, id, func(tk *task.Task) error {
		tk.Specification = "spec"
		tk.DoneWhen = "criteria"
		tk.Status.Specification = task.StageStatus{State: task.StageDone}
		tk.State = task.StateStarted
		return nil
	}); err != nil {
		t.Fatalf("specify: %v", err)
	}
	if _, err := task.MutateTask(tasksDir, id, func(tk *task.Task) error {
		tk.Status.Implementation = task.StageStatus{State: task.StageDone}
		return nil
	}); err != nil {
		t.Fatalf("implement: %v", err)
	}
}

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func recordReview(t *testing.T, tasksDir, id string, approved bool, findings []task.Finding) task.Task {
	t.Helper()
	got, err := task.MutateTask(tasksDir, id, func(tk *task.Task) error {
		attempt := len(tk.Reviews) + 1
		tk.Reviews = append(
			tk.Reviews,
			task.Review{Attempt: attempt, Approved: approved, Findings: findings, CreatedAt: task.Now()},
		)
		tk.Status.Verification.Attempts = attempt
		if approved {
			now := task.Now()
			tk.Status.Verification.State = task.StageDone
			tk.Status.Verification.CompletedAt = &now
			tk.Status.Review.State = task.StagePending
		} else if attempt >= 2 {
			tk.State = task.StateBlocked
			tk.Blocked = &task.Blocked{Stage: task.StageVerification, Reason: "second rejection", At: task.Now()}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("record review: %v", err)
	}
	return got
}

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func verify(t *testing.T, tasksDir, id string, checks map[string]task.CheckResult, output string) {
	t.Helper()
	if _, err := task.MutateTask(tasksDir, id, func(tk *task.Task) error {
		tk.Verifications = append(
			tk.Verifications,
			task.Verification{Checks: checks, Output: output, CreatedAt: task.Now()},
		)
		return nil
	}); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestNextSpecifyThenImplement(t *testing.T) {
	dir := t.TempDir()
	newTestTask(t, dir, "abc")

	g, err := Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || !strings.Contains(g.Message, "specification") {
		t.Fatalf("next at creation = %+v, want dispatch specify", g)
	}
	if g.ReportWith != "specify abc" {
		t.Errorf("report_with = %q, want task specify abc", g.ReportWith)
	}

	readyForVerify(t, dir, "abc")
	g, err = Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionRun || g.ReportWith != "verify abc" {
		t.Fatalf("next after specify+implement = %+v, want run verify", g)
	}
}

func TestNextVerificationLoop(t *testing.T) {
	dir := t.TempDir()
	newTestTask(t, dir, "abc")
	readyForVerify(t, dir, "abc")

	verify(t, dir, "abc", map[string]task.CheckResult{"vet": task.CheckError}, "boom")
	g, err := Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || !strings.Contains(g.Message, "boom") {
		t.Fatalf("next after failed verify = %+v, want dispatch fix mentioning failure", g)
	}

	verify(t, dir, "abc", map[string]task.CheckResult{"vet": task.CheckOK}, "")
	g, err = Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "review record abc" {
		t.Fatalf("next after passing verify = %+v, want dispatch review", g)
	}

	recordReview(t, dir, "abc", false, []task.Finding{{File: "x.go", Detail: "bug"}})
	g, err = Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "verify abc" || !strings.Contains(g.Message, "bug") {
		t.Fatalf("next after rejected review = %+v, want dispatch fix mentioning findings", g)
	}
}

func TestNextDraftCommitThenWaitHumanReview(t *testing.T) {
	dir := t.TempDir()
	newTestTask(t, dir, "abc")
	readyForVerify(t, dir, "abc")
	verify(t, dir, "abc", map[string]task.CheckResult{"vet": task.CheckOK}, "")
	recordReview(t, dir, "abc", true, nil)

	g, err := Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "commit abc" {
		t.Fatalf("next after verification passes = %+v, want dispatch commit", g)
	}

	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Git.Commit = &task.GitCommit{Hash: "deadbeef", Message: "docs: x", Type: "docs", At: time.Now().UTC()}
		return nil
	}); err != nil {
		t.Fatalf("simulate commit: %v", err)
	}

	g, err = Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionWait || g.ReportWith != "" {
		t.Fatalf("next after commit recorded = %+v, want wait", g)
	}
	if !strings.Contains(g.Message, "deadbeef") {
		t.Errorf("wait message = %q, want it to mention the commit hash", g.Message)
	}
}

// TestNextAutoApproveSkipsHumanReview covers a task created with
// --auto-approve (task.AutoApprove): commit.Commit completes its review
// stage directly (no HumanReviews entry - see commit.go), so by the time a
// fresh commit is recorded, task next should already see review done and
// move straight to the merge stage rather than waiting for a human.
func TestNextAutoApproveSkipsHumanReview(t *testing.T) {
	dir := t.TempDir()
	newTestTask(t, dir, "abc")
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.AutoApprove = true
		return nil
	}); err != nil {
		t.Fatalf("set auto-approve: %v", err)
	}
	readyForVerify(t, dir, "abc")
	verify(t, dir, "abc", map[string]task.CheckResult{"vet": task.CheckOK}, "")
	recordReview(t, dir, "abc", true, nil)

	g, err := Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "commit abc" {
		t.Fatalf("next after verification passes = %+v, want dispatch commit", g)
	}

	// Simulate what commit.Commit does for an auto-approve task: the commit
	// is recorded and review completes in the same step, with no
	// HumanReviews entry - no human reviewed this.
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Git.Commit = &task.GitCommit{Hash: "deadbeef", Message: "docs: x", Type: "docs", At: time.Now().UTC()}
		tk.Status.Review = task.StageStatus{State: task.StageDone, CompletedAt: new(task.Now())}
		return nil
	}); err != nil {
		t.Fatalf("simulate commit: %v", err)
	}

	g, err = Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionRun || g.ReportWith != "merge abc" {
		t.Fatalf("next after auto-approve commit = %+v, want run merge (no human-review wait)", g)
	}
}

func TestNextReviewRejectRecoverySkipsAutomatedReview(t *testing.T) {
	dir := t.TempDir()
	newTestTask(t, dir, "abc")
	readyForVerify(t, dir, "abc")
	verify(t, dir, "abc", map[string]task.CheckResult{"vet": task.CheckOK}, "")
	recordReview(t, dir, "abc", true, nil)
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Git.Commit = &task.GitCommit{Hash: "aaa111", Message: "docs: x", Type: "docs", At: time.Now().UTC()}
		return nil
	}); err != nil {
		t.Fatalf("simulate commit: %v", err)
	}

	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.HumanReviews = append(
			tk.HumanReviews,
			task.HumanReview{Approved: false, Comment: "needs more detail", At: task.Now()},
		)
		tk.Status.Review.State = task.StageInProgress
		return nil
	}); err != nil {
		t.Fatalf("reject: %v", err)
	}

	g, err := Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "verify abc" ||
		!strings.Contains(g.Message, "needs more detail") {
		t.Fatalf("next after human rejection = %+v, want dispatch fix mentioning the human's reason", g)
	}

	verify(t, dir, "abc", map[string]task.CheckResult{"vet": task.CheckOK}, "")
	g, err = Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "commit abc" {
		t.Fatalf(
			"next after clean re-verify during recovery = %+v, want dispatch a NEW commit, not automated review",
			g,
		)
	}
}

func TestNextMergeAndDone(t *testing.T) {
	dir := t.TempDir()
	newTestTask(t, dir, "abc")
	readyForVerify(t, dir, "abc")
	verify(t, dir, "abc", map[string]task.CheckResult{"vet": task.CheckOK}, "")
	recordReview(t, dir, "abc", true, nil)
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Git.Commit = &task.GitCommit{Hash: "aaa", Message: "docs: x", At: time.Now().UTC()}
		return nil
	}); err != nil {
		t.Fatalf("simulate commit: %v", err)
	}
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Status.Review.State = task.StageDone
		tk.HumanReviews = append(tk.HumanReviews, task.HumanReview{Approved: true, Comment: "LGTM", At: task.Now()})
		return nil
	}); err != nil {
		t.Fatalf("approve: %v", err)
	}

	g, err := Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionRun || g.ReportWith != "merge abc" {
		t.Fatalf("next after approval = %+v, want run merge", g)
	}

	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Status.Merge = task.StageStatus{State: task.StageDone}
		tk.State = task.StateCompleted
		return nil
	}); err != nil {
		t.Fatalf("merge: %v", err)
	}
	g, err = Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDone {
		t.Fatalf("next after merge = %+v, want done", g)
	}
	if strings.Contains(g.Message, "nothing to merge") || strings.Contains(g.Message, "no merge was needed") {
		t.Fatalf("next after non-trunk merge = %q, should not claim trunk mode", g.Message)
	}
}

// TestNextMergeTrunk mirrors TestNextMergeAndDone for a task created with
// --trunk (task.Git.Trunk): the merge-stage guidance should say there's
// nothing to actually merge, and the completion message shouldn't claim a
// merge happened - regressions for the wrong-branch/"merged into main"
// wording task next used to have here.
func TestNextMergeTrunk(t *testing.T) {
	dir := t.TempDir()
	newTestTask(t, dir, "abc")
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Git.Worktree = "/repo"
		tk.Git.Branch = "dev"
		tk.Git.Trunk = true
		return nil
	}); err != nil {
		t.Fatalf("set trunk: %v", err)
	}
	readyForVerify(t, dir, "abc")
	verify(t, dir, "abc", map[string]task.CheckResult{"vet": task.CheckOK}, "")
	recordReview(t, dir, "abc", true, nil)
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Git.Commit = &task.GitCommit{Hash: "aaa", Message: "docs: x", At: time.Now().UTC()}
		return nil
	}); err != nil {
		t.Fatalf("simulate commit: %v", err)
	}
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Status.Review.State = task.StageDone
		tk.HumanReviews = append(tk.HumanReviews, task.HumanReview{Approved: true, Comment: "LGTM", At: task.Now()})
		return nil
	}); err != nil {
		t.Fatalf("approve: %v", err)
	}

	g, err := Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionRun || g.ReportWith != "merge abc" {
		t.Fatalf("next after approval = %+v, want run merge", g)
	}
	if !strings.Contains(g.Message, "nothing to merge") {
		t.Fatalf("next merge guidance for trunk task = %q, want it to say there's nothing to merge", g.Message)
	}
	if strings.Contains(g.Message, "into the base branch") {
		t.Fatalf("next merge guidance for trunk task = %q, should not tell it to merge into a base branch", g.Message)
	}

	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.Status.Merge = task.StageStatus{State: task.StageDone}
		tk.State = task.StateCompleted
		return nil
	}); err != nil {
		t.Fatalf("merge: %v", err)
	}
	g, err = Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDone {
		t.Fatalf("next after merge = %+v, want done", g)
	}
	if !strings.Contains(g.Message, "already on branch dev") {
		t.Fatalf("next done message for trunk task = %q, want it to say already on branch dev", g.Message)
	}
	if strings.Contains(g.Message, "merged into main") {
		t.Fatalf("next done message for trunk task = %q, should not claim a merge into main", g.Message)
	}
}

func TestNextBlockedAndFailed(t *testing.T) {
	dir := t.TempDir()
	newTestTask(t, dir, "abc")
	if _, err := task.MutateTask(dir, "abc", func(tk *task.Task) error {
		tk.State = task.StateBlocked
		tk.Blocked = &task.Blocked{Stage: task.StageImplementation, Reason: "stuck", At: task.Now()}
		return nil
	}); err != nil {
		t.Fatalf("escalate: %v", err)
	}
	g, err := Next(dir, Request{ID: "abc"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionWait || !strings.Contains(g.Message, "stuck") {
		t.Fatalf("next on blocked task = %+v, want wait mentioning the reason", g)
	}

	newTestTask(t, dir, "def")
	if _, err := task.MutateTask(dir, "def", func(tk *task.Task) error {
		tk.State = task.StateFailed
		tk.FailureReason = "no longer relevant"
		return nil
	}); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	g, err = Next(dir, Request{ID: "def"})
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDone || !strings.Contains(g.Message, "no longer relevant") {
		t.Fatalf("next on abandoned task = %+v, want done mentioning the reason", g)
	}
}
