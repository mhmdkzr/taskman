package task

import (
	"strings"
	"testing"
	"time"
)

func TestNextSpecifyThenImplement(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")

	g, err := Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || !strings.Contains(g.Message, "specification") {
		t.Fatalf("next at creation = %+v, want dispatch specify", g)
	}
	if g.ReportWith != "task specify abc" {
		t.Errorf("report_with = %q, want task specify abc", g.ReportWith)
	}

	if _, err := Specify(repo, "abc", SpecifyRequest{Result: "spec", DoneWhen: "criteria"}); err != nil {
		t.Fatalf("specify: %v", err)
	}
	g, err = Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "task implement abc" {
		t.Fatalf("next after specify = %+v, want dispatch implement", g)
	}
}

func TestNextVerificationLoop(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")

	g, err := Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionRun || g.ReportWith != "task verify abc" {
		t.Fatalf("next before any verify = %+v, want run verify", g)
	}

	if _, err := Verify(
		repo,
		"abc",
		VerifyRequest{Checks: map[string]CheckResult{"vet": CheckError}, Output: "boom"},
	); err != nil {
		t.Fatalf("verify: %v", err)
	}
	g, err = Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || !strings.Contains(g.Message, "boom") {
		t.Fatalf("next after failed verify = %+v, want dispatch fix mentioning failure", g)
	}

	if _, err := Verify(repo, "abc", VerifyRequest{Checks: map[string]CheckResult{"vet": CheckOK}}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	g, err = Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "task review record abc" {
		t.Fatalf("next after passing verify = %+v, want dispatch review", g)
	}

	if _, err := RecordReview(
		repo,
		"abc",
		ReviewRecordRequest{Approved: false, Findings: []Finding{{File: "x.go", Detail: "bug"}}},
	); err != nil {
		t.Fatalf("record review: %v", err)
	}
	g, err = Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "task verify abc" || !strings.Contains(g.Message, "bug") {
		t.Fatalf("next after rejected review = %+v, want dispatch fix mentioning findings", g)
	}
}

func TestNextDraftCommitThenWaitHumanReview(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")
	if _, err := RecordReview(repo, "abc", ReviewRecordRequest{Approved: true}); err != nil {
		t.Fatalf("record review: %v", err)
	}

	g, err := Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "task commit abc" {
		t.Fatalf("next after verification passes = %+v, want dispatch commit", g)
	}

	if _, err := repo.Mutate("abc", func(tk *Task) error {
		tk.Git.Commit = &GitCommit{Hash: "deadbeef", Message: "docs: x", Type: "docs", At: time.Now().UTC()}
		return nil
	}); err != nil {
		t.Fatalf("simulate commit: %v", err)
	}

	g, err = Next(repo, "abc")
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

func TestNextReviewRejectRecoverySkipsAutomatedReview(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")
	if _, err := RecordReview(repo, "abc", ReviewRecordRequest{Approved: true}); err != nil {
		t.Fatalf("record review: %v", err)
	}
	commitAt := time.Now().UTC()
	if _, err := repo.Mutate("abc", func(tk *Task) error {
		tk.Git.Commit = &GitCommit{Hash: "aaa111", Message: "docs: x", Type: "docs", At: commitAt}
		return nil
	}); err != nil {
		t.Fatalf("simulate commit: %v", err)
	}

	if _, err := RejectReview(repo, "abc", "needs more detail"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	g, err := Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "task verify abc" ||
		!strings.Contains(g.Message, "needs more detail") {
		t.Fatalf("next after human rejection = %+v, want dispatch fix mentioning the human's reason", g)
	}

	if _, err := Verify(repo, "abc", VerifyRequest{Checks: map[string]CheckResult{"vet": CheckOK}}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	g, err = Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDispatch || g.ReportWith != "task commit abc" {
		t.Fatalf(
			"next after clean re-verify during recovery = %+v, want dispatch a NEW commit, not automated review",
			g,
		)
	}

	if _, err := repo.Mutate("abc", func(tk *Task) error {
		tk.Git.Commit = &GitCommit{Hash: "bbb222", Message: "docs: y", Type: "docs", At: time.Now().UTC()}
		return nil
	}); err != nil {
		t.Fatalf("simulate second commit: %v", err)
	}
	got, err := repo.Get("abc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.Review.State != StageInProgress {
		t.Fatalf("review.state = %v, want still in_progress until task commit's effect runs", got.Status.Review.State)
	}
}

func TestNextMergeAndDone(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")
	if _, err := RecordReview(repo, "abc", ReviewRecordRequest{Approved: true}); err != nil {
		t.Fatalf("record review: %v", err)
	}
	if _, err := repo.Mutate("abc", func(tk *Task) error {
		tk.Git.Commit = &GitCommit{Hash: "aaa", Message: "docs: x", At: time.Now().UTC()}
		return nil
	}); err != nil {
		t.Fatalf("simulate commit: %v", err)
	}
	if _, err := ApproveReview(repo, "abc", "LGTM"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	g, err := Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionRun || g.ReportWith != "task merge abc" {
		t.Fatalf("next after approval = %+v, want run merge", g)
	}

	if _, err := Merge(repo, "abc", MergeRequest{}); err != nil {
		t.Fatalf("merge: %v", err)
	}
	g, err = Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDone {
		t.Fatalf("next after merge = %+v, want done", g)
	}
}

func TestNextBlockedAndFailed(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	if _, err := Escalate(repo, "abc", EscalateRequest{Stage: StageImplementation, Reason: "stuck"}); err != nil {
		t.Fatalf("escalate: %v", err)
	}
	g, err := Next(repo, "abc")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionWait || !strings.Contains(g.Message, "stuck") {
		t.Fatalf("next on blocked task = %+v, want wait mentioning the reason", g)
	}

	newTestTask(t, repo, "def")
	if _, err := Abandon(repo, "def", "no longer relevant"); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	g, err = Next(repo, "def")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if g.Action != ActionDone || !strings.Contains(g.Message, "no longer relevant") {
		t.Fatalf("next on abandoned task = %+v, want done mentioning the reason", g)
	}
}
