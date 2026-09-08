package task

import (
	"errors"
	"testing"
)

func newTestTask(t *testing.T, repo *Repo, id string) {
	t.Helper()
	tk := Task{
		ID:         id,
		State:      StateCreated,
		Title:      "Test",
		Definition: "def",
		Status: Status{
			Definition:     StageStatus{State: StageDone},
			Specification:  StageStatus{State: StagePending},
			Implementation: StageStatus{State: StagePending},
			Verification:   StageStatus{State: StagePending},
			Review:         StageStatus{State: StagePending},
			Merge:          StageStatus{State: StagePending},
		},
		Git: Git{Worktree: "/tmp/wt", Branch: "task/" + id},
	}
	if err := repo.Create(tk); err != nil {
		t.Fatalf("create: %v", err)
	}
}

func TestSpecifyThenImplement(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")

	got, err := Specify(repo, "abc", SpecifyRequest{Result: "spec", DoneWhen: "criteria"})
	if err != nil {
		t.Fatalf("specify: %v", err)
	}
	if got.Status.Specification.State != StageDone || got.State != StateStarted {
		t.Fatalf("after specify: status=%+v state=%v", got.Status.Specification, got.State)
	}

	if _, err := Specify(repo, "abc", SpecifyRequest{}); err == nil {
		t.Error("re-specify: want precondition error, got nil")
	}

	if _, err := Implement(repo, "abc"); err != nil {
		t.Fatalf("implement: %v", err)
	}
	got, _ = repo.Get("abc")
	if got.Status.Implementation.State != StageDone {
		t.Errorf("implementation state = %v, want done", got.Status.Implementation.State)
	}
}

func TestImplementRequiresSpecification(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	if _, err := Implement(repo, "abc"); err == nil {
		t.Error("implement before specify: want error, got nil")
	}
}

func readyForVerify(t *testing.T, repo *Repo, id string) {
	t.Helper()
	if _, err := Specify(repo, id, SpecifyRequest{Result: "spec", DoneWhen: "criteria"}); err != nil {
		t.Fatalf("specify: %v", err)
	}
	if _, err := Implement(repo, id); err != nil {
		t.Fatalf("implement: %v", err)
	}
}

func TestVerifyAppendsAndPassed(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")

	got, err := Verify(repo, "abc", VerifyRequest{Checks: map[string]CheckResult{"vet": CheckError}, Output: "bad"})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(got.Verifications) != 1 || got.Verifications[0].Passed() {
		t.Fatalf("verifications = %+v, want one failing entry", got.Verifications)
	}
	if got.Status.Verification.State == StageDone {
		t.Error("verify alone must never set verification.state: done")
	}

	got, err = Verify(repo, "abc", VerifyRequest{Checks: map[string]CheckResult{"vet": CheckOK}})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(got.Verifications) != 2 || !got.Verifications[1].Passed() {
		t.Fatalf("verifications = %+v, want second entry passing", got.Verifications)
	}
}

func TestRecordReviewApprovedAdvances(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")

	got, err := RecordReview(repo, "abc", ReviewRecordRequest{Approved: true})
	if err != nil {
		t.Fatalf("record review: %v", err)
	}
	if got.Status.Verification.State != StageDone {
		t.Errorf("verification.state = %v, want done", got.Status.Verification.State)
	}
	if got.Status.Review.State != StagePending {
		t.Errorf("review.state = %v, want pending", got.Status.Review.State)
	}
	if got.Status.Verification.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", got.Status.Verification.Attempts)
	}
}

func TestRecordReviewTwoRoundCapBlocks(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")

	got, err := RecordReview(
		repo,
		"abc",
		ReviewRecordRequest{Approved: false, Findings: []Finding{{File: "x.go", Detail: "bug"}}},
	)
	if err != nil {
		t.Fatalf("record review 1: %v", err)
	}
	if got.State == StateBlocked {
		t.Fatal("first rejection must not block the task")
	}
	if got.Status.Verification.State == StageDone {
		t.Fatal("rejected review must not mark verification done")
	}

	got, err = RecordReview(
		repo,
		"abc",
		ReviewRecordRequest{Approved: false, Findings: []Finding{{File: "x.go", Detail: "still buggy"}}},
	)
	if err != nil {
		t.Fatalf("record review 2: %v", err)
	}
	if got.State != StateBlocked {
		t.Fatalf("state = %v, want blocked after second rejection", got.State)
	}
	if got.Blocked == nil || got.Blocked.Stage != StageVerification {
		t.Fatalf("blocked = %+v, want stage=verification", got.Blocked)
	}
}

func TestVerifyAndRecordReviewRejectBlockedFailed(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")
	if _, err := Escalate(repo, "abc", EscalateRequest{Stage: StageImplementation, Reason: "gave up"}); err != nil {
		t.Fatalf("escalate: %v", err)
	}

	if _, err := Verify(repo, "abc", VerifyRequest{Checks: map[string]CheckResult{"vet": CheckOK}}); err == nil {
		t.Error("verify on blocked task: want error, got nil")
	}
	if _, err := RecordReview(repo, "abc", ReviewRecordRequest{Approved: true}); err == nil {
		t.Error("record review on blocked task: want error, got nil")
	}
}

func TestEscalateRefusesTerminal(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	if _, err := Abandon(repo, "abc", "done with it"); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	if _, err := Escalate(repo, "abc", EscalateRequest{Stage: StageImplementation, Reason: "x"}); err == nil {
		t.Error("escalate on failed task: want error, got nil")
	}
}

func TestApproveRejectReviewAndRecovery(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")
	if _, err := RecordReview(repo, "abc", ReviewRecordRequest{Approved: true}); err != nil {
		t.Fatalf("record review: %v", err)
	}

	if _, err := ApproveReview(repo, "abc", "not yet, need a real commit test"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got, _ := repo.Get("abc")
	if got.Status.Review.State != StageDone {
		t.Fatalf("review.state = %v, want done", got.Status.Review.State)
	}
	if len(got.HumanReviews) != 1 || !got.HumanReviews[0].Approved {
		t.Fatalf("human reviews = %+v, want one approval", got.HumanReviews)
	}

	if _, err := ApproveReview(repo, "abc", ""); err == nil {
		t.Error("approve when already done: want error, got nil")
	}
}

func TestRejectReviewStartsRecovery(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	readyForVerify(t, repo, "abc")
	if _, err := RecordReview(repo, "abc", ReviewRecordRequest{Approved: true}); err != nil {
		t.Fatalf("record review: %v", err)
	}

	got, err := RejectReview(repo, "abc", "please add tests")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if got.Status.Review.State != StageInProgress {
		t.Fatalf("review.state = %v, want in_progress", got.Status.Review.State)
	}
	if len(got.HumanReviews) != 1 || got.HumanReviews[0].Approved || got.HumanReviews[0].Comment != "please add tests" {
		t.Fatalf("human reviews = %+v", got.HumanReviews)
	}

	if _, err := RejectReview(repo, "abc", "again"); err == nil {
		t.Error("reject while already in_progress: want error, got nil")
	}
}

func TestMergeRequiresReviewDone(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	if _, err := Merge(repo, "abc", MergeRequest{}); err == nil {
		t.Error("merge before review done: want error, got nil")
	}

	readyForVerify(t, repo, "abc")
	if _, err := RecordReview(repo, "abc", ReviewRecordRequest{Approved: true}); err != nil {
		t.Fatalf("record review: %v", err)
	}
	if _, err := ApproveReview(repo, "abc", "LGTM"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got, err := Merge(repo, "abc", MergeRequest{Commit: "deadbeef"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if got.State != StateCompleted || got.Status.Merge.State != StageDone {
		t.Fatalf("after merge: state=%v merge=%+v", got.State, got.Status.Merge)
	}
}

func TestAbandonTerminal(t *testing.T) {
	repo := NewRepo(t.TempDir())
	newTestTask(t, repo, "abc")
	got, err := Abandon(repo, "abc", "no longer needed")
	if err != nil {
		t.Fatalf("abandon: %v", err)
	}
	if got.State != StateFailed || got.FailureReason != "no longer needed" {
		t.Fatalf("after abandon: state=%v reason=%q", got.State, got.FailureReason)
	}
	if _, err := Abandon(repo, "abc", "again"); err != nil {
		t.Errorf("abandon an already-failed task should be idempotent, got: %v", err)
	}

	repo2 := NewRepo(t.TempDir())
	newTestTask(t, repo2, "def")
	readyForVerify(t, repo2, "def")
	if _, err := RecordReview(repo2, "def", ReviewRecordRequest{Approved: true}); err != nil {
		t.Fatalf("record review: %v", err)
	}
	if _, err := ApproveReview(repo2, "def", ""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err := Merge(repo2, "def", MergeRequest{}); err != nil {
		t.Fatalf("merge: %v", err)
	}
	if _, err := Abandon(repo2, "def", "x"); err == nil {
		t.Error("abandon a completed task: want error, got nil")
	}
}

func TestValidateLabels(t *testing.T) {
	if err := ValidateLabels(map[string]string{"priority": "high", "type": "anything-goes"}); err != nil {
		t.Errorf("valid labels: %v", err)
	}
	err := ValidateLabels(map[string]string{"priority": "urgent"})
	if !errors.Is(err, ErrInvalidLabel) {
		t.Errorf("invalid priority: err = %v, want ErrInvalidLabel", err)
	}
}
