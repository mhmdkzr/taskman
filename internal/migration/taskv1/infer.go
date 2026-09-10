package taskv1

import (
	"fmt"
	"time"

	"github.com/mhmdkzr/taskman/internal/task"
)

func (s legacyStatus) present() bool {
	stages := [...]legacyStageStatus{
		s.Definition, s.Specification, s.Implementation, s.Verification, s.Review, s.Merge,
	}
	for _, stage := range stages {
		if stage.State != "" || stage.CompletedAt != nil || stage.Attempts != 0 {
			return true
		}
	}
	return false
}

func convert(old legacyTask) (task.Task, error) {
	state, resume, err := inferState(old)
	if err != nil {
		return task.Task{}, err
	}
	converted := task.Task{
		ID: old.ID, State: state, Title: old.Title, Labels: old.Labels,
		Definition: old.Definition, Specification: old.Specification, DoneWhen: old.DoneWhen,
		References: old.References, Git: old.Git, Verifications: old.Verifications,
		Reviews: old.Reviews, HumanReviews: old.HumanReviews,
		FailureReason: old.FailureReason, AutoApprove: old.AutoApprove,
	}
	if state == task.StateBlocked {
		if old.Blocked == nil {
			return task.Task{}, fmt.Errorf("legacy blocked task has no blocked details")
		}
		converted.Blocked = &task.Blocked{
			ResumeState: resume, Stage: old.Blocked.Stage, Reason: old.Blocked.Reason, At: old.Blocked.At,
		}
	}
	if err := task.Validate(converted); err != nil {
		return task.Task{}, fmt.Errorf("validate converted task: %w", err)
	}
	return converted, nil
}

func inferState(old legacyTask) (task.State, task.State, error) {
	switch old.State {
	case "completed":
		return task.StateCompleted, "", nil
	case "failed":
		return task.StateAbandoned, "", nil
	case "blocked":
		copy := old
		copy.State = "started"
		resume, _, err := inferState(copy)
		if err != nil {
			return "", "", fmt.Errorf("infer blocked resume state: %w", err)
		}
		return task.StateBlocked, resume, nil
	case "created", "started":
		return inferActiveState(old)
	default:
		return "", "", fmt.Errorf("unknown legacy task state %q", old.State)
	}
}

func inferActiveState(old legacyTask) (task.State, task.State, error) {
	if old.Status.Specification.State != "done" {
		return task.StateSpecify, "", nil
	}
	if old.Status.Implementation.State != "done" {
		return task.StateImplement, "", nil
	}
	if old.Status.Verification.State != "done" {
		lastVerification := lastVerification(old)
		if lastVerification == nil {
			return task.StateVerify, "", nil
		}
		if !lastVerification.Passed() {
			return task.StateFixVerificationFailure, "", nil
		}
		lastReview := lastReview(old)
		if lastReview == nil || lastReview.CreatedAt.Before(lastVerification.CreatedAt) {
			return task.StateAutomatedReview, "", nil
		}
		if lastReview.Approved {
			return "", "", fmt.Errorf("approved automated review left verification pending")
		}
		return task.StateFixAutomatedReviewFindings, "", nil
	}

	switch old.Status.Review.State {
	case "in_progress":
		lastHuman := lastHumanReview(old)
		if lastHuman == nil {
			return "", "", fmt.Errorf("review recovery has no human rejection")
		}
		lastVerification := lastVerification(old)
		if lastVerification == nil || !lastVerification.CreatedAt.After(lastHuman.At) || !lastVerification.Passed() {
			return task.StateFixHumanReviewFindings, "", nil
		}
		if hasCommitSince(old, &lastHuman.At) {
			return task.StateHumanReview, "", nil
		}
		return task.StateCommit, "", nil
	case "done":
		if old.Status.Merge.State == "done" {
			return task.StateCompleted, "", nil
		}
		return task.StateMerge, "", nil
	default:
		if hasCommitSince(old, old.Status.Verification.CompletedAt) {
			return task.StateHumanReview, "", nil
		}
		return task.StateCommit, "", nil
	}
}

func hasCommitSince(old legacyTask, since *time.Time) bool {
	return old.Git.Commit != nil && since != nil && !old.Git.Commit.At.Before(*since)
}

func lastVerification(old legacyTask) *task.Verification {
	if len(old.Verifications) == 0 {
		return nil
	}
	return &old.Verifications[len(old.Verifications)-1]
}

func lastReview(old legacyTask) *task.Review {
	if len(old.Reviews) == 0 {
		return nil
	}
	return &old.Reviews[len(old.Reviews)-1]
}

func lastHumanReview(old legacyTask) *task.HumanReview {
	if len(old.HumanReviews) == 0 {
		return nil
	}
	return &old.HumanReviews[len(old.HumanReviews)-1]
}
