package task

import (
	"context"
	"fmt"
	"strings"
)

// SpecifyRequest is task specify's input.
type SpecifyRequest struct {
	Result   string
	DoneWhen string
}

// Specify writes a task's specification and done_when, drafted from its
// definition - design.md §6.
func Specify(repo *Repo, id string, req SpecifyRequest) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.Status.Specification.State == StageDone {
			return notInState("specification", string(t.Status.Specification.State), "!= done")
		}
		t.Specification = req.Result
		t.DoneWhen = req.DoneWhen
		t.Status.Specification = StageStatus{State: StageDone, CompletedAt: new(now())}
		t.State = StateStarted
		return nil
	})
}

// Implement marks a task's implementation attempt as done - the diff lives
// in the worktree, not the task file.
func Implement(repo *Repo, id string) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.Status.Specification.State != StageDone {
			return notInState("specification", string(t.Status.Specification.State), "done")
		}
		if t.Status.Implementation.State == StageDone {
			return notInState("implementation", string(t.Status.Implementation.State), "!= done")
		}
		t.Status.Implementation = StageStatus{State: StageDone, CompletedAt: new(now())}
		return nil
	})
}

// VerifyRequest is task verify's input - one reported build-check attempt.
type VerifyRequest struct {
	Checks map[string]CheckResult
	Output string
}

// Verify appends one build-check attempt to a task's verifications log -
// design.md §6. It never sets verification.state itself; that only happens
// via RecordReview's approval.
func Verify(repo *Repo, id string, req VerifyRequest) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.Status.Implementation.State != StageDone {
			return notInState("implementation", string(t.Status.Implementation.State), "done")
		}
		if err := notBlockedOrFailed(t); err != nil {
			return err
		}
		t.Verifications = append(t.Verifications, Verification{
			Checks:    req.Checks,
			Output:    req.Output,
			CreatedAt: now(),
		})
		return nil
	})
}

// ReviewRecordRequest is task review record's input - one automated review
// round's verdict.
type ReviewRecordRequest struct {
	Approved bool
	Findings []Finding
}

// RecordReview reports the automated review round's verdict inside
// verification - design.md §6's verification stage. Approval advances the
// task to the review stage; rejection either loops back for another
// attempt (attempt 1) or blocks the task (attempt 2, the fixed two-round
// cap).
func RecordReview(repo *Repo, id string, req ReviewRecordRequest) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.Status.Implementation.State != StageDone {
			return notInState("implementation", string(t.Status.Implementation.State), "done")
		}
		if err := notBlockedOrFailed(t); err != nil {
			return err
		}

		attempt := len(t.Reviews) + 1
		t.Reviews = append(t.Reviews, Review{
			Attempt:   attempt,
			Approved:  req.Approved,
			Findings:  req.Findings,
			CreatedAt: now(),
		})
		t.Status.Verification.Attempts = attempt

		if req.Approved {
			t.Status.Verification.State = StageDone
			t.Status.Verification.CompletedAt = new(now())
			t.Status.Review.State = StagePending
			return nil
		}

		if attempt >= 2 {
			t.State = StateBlocked
			t.Blocked = &Blocked{
				Stage:  StageVerification,
				Reason: fmt.Sprintf("Second review rejected: %s", summarizeFindings(req.Findings)),
				At:     now(),
			}
		}
		return nil
	})
}

// CommitRequest is task commit's input - just which commit to read, if not
// HEAD.
type CommitRequest struct {
	Commit string
}

// Commit reads the caller's already-made commit directly out of the
// worktree via git log, rather than trusting reported text - design.md §6
// "When the commit happens". Called once right after verification first
// passes, and again each time a review-reject-recovery cycle clears.
func Commit(ctx context.Context, repo *Repo, git *GitClient, id string, req CommitRequest) (Task, error) {
	t, err := repo.Get(id)
	if err != nil {
		return Task{}, err
	}
	if t.Status.Verification.State != StageDone {
		return Task{}, notInState("verification", string(t.Status.Verification.State), "done")
	}
	commit, err := git.ReadCommit(ctx, t.Git.Worktree, req.Commit)
	if err != nil {
		return Task{}, fmt.Errorf("read commit: %w", err)
	}
	commit.At = now()
	return repo.Mutate(id, func(t *Task) error {
		if t.Status.Verification.State != StageDone {
			return notInState("verification", string(t.Status.Verification.State), "done")
		}
		t.Git.Commit = &commit
		t.Status.Review.State = StagePending
		return nil
	})
}

// EscalateRequest is task escalate's input.
type EscalateRequest struct {
	Stage  string
	Reason string
}

// Escalate blocks a task on a caller's own report that a dispatched agent
// gave up rather than keep iterating - design.md §6's "Giving up".
func Escalate(repo *Repo, id string, req EscalateRequest) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.State == StateCompleted || t.State == StateFailed {
			return notInState("task", string(t.State), "not already terminal")
		}
		t.State = StateBlocked
		t.Blocked = &Blocked{Stage: req.Stage, Reason: req.Reason, At: now()}
		return nil
	})
}

// ApproveReview records a human's approval at the review stage.
func ApproveReview(repo *Repo, id string, comment string) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.Status.Review.State != StagePending {
			return notInState("review", string(t.Status.Review.State), "pending")
		}
		t.Status.Review.State = StageDone
		t.Status.Review.CompletedAt = new(now())
		t.HumanReviews = append(t.HumanReviews, HumanReview{Approved: true, Comment: comment, At: now()})
		return nil
	})
}

// RejectReview records a human's rejection and starts review-reject
// recovery - design.md §6.
func RejectReview(repo *Repo, id string, reason string) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.Status.Review.State != StagePending {
			return notInState("review", string(t.Status.Review.State), "pending")
		}
		t.HumanReviews = append(t.HumanReviews, HumanReview{Approved: false, Comment: reason, At: now()})
		t.Status.Review.State = StageInProgress
		return nil
	})
}

// MergeRequest is task merge's input.
type MergeRequest struct {
	Commit string
}

// Merge records that the caller already merged the task's branch.
func Merge(repo *Repo, id string, req MergeRequest) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.Status.Review.State != StageDone {
			return notInState("review", string(t.Status.Review.State), "done")
		}
		t.Status.Merge = StageStatus{State: StageDone, CompletedAt: new(now())}
		t.State = StateCompleted
		if req.Commit != "" && t.Git.Commit != nil {
			t.Git.Commit.Hash = req.Commit
		}
		return nil
	})
}

// Abandon marks a task failed for good.
func Abandon(repo *Repo, id string, reason string) (Task, error) {
	return repo.Mutate(id, func(t *Task) error {
		if t.State == StateCompleted {
			return notInState("task", string(t.State), "not already completed")
		}
		t.State = StateFailed
		t.FailureReason = reason
		return nil
	})
}

func notInState(stage, have, want string) error {
	return &InvalidTransitionError{Stage: stage, Have: have, Want: want}
}

func notBlockedOrFailed(t *Task) error {
	if t.State == StateBlocked || t.State == StateFailed {
		return &InvalidTransitionError{Stage: "task", Have: string(t.State), Want: "not blocked/failed"}
	}
	return nil
}

func summarizeFindings(findings []Finding) string {
	if len(findings) == 0 {
		return "no findings reported"
	}
	parts := make([]string, len(findings))
	for i, f := range findings {
		parts[i] = fmt.Sprintf("%s: %s", f.File, f.Detail)
	}
	return strings.Join(parts, "; ")
}
