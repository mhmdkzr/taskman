package task

import (
	"fmt"
	"maps"
)

// Apply evaluates one event against the compiled workflow. It never mutates
// current, even when the returned transition fails validation.
func Apply(current Task, event Event) (Task, error) {
	if event == nil {
		return Task{}, fmt.Errorf("apply event: event is required")
	}
	state, ok := workflow.states[current.State]
	if !ok {
		return Task{}, fmt.Errorf("apply event: %w: %q", ErrUnknownState, current.State)
	}
	if state.terminal {
		return Task{}, newInvalidTransition(current.State, event.eventKind())
	}

	transition, ok := workflow.global[event.eventKind()]
	if !ok {
		transition, ok = state.on[event.eventKind()]
	}
	if !ok {
		return Task{}, newInvalidTransition(current.State, event.eventKind())
	}
	if transition.guard != nil && !transition.guard(current, event) {
		return Task{}, newInvalidTransition(current.State, event.eventKind())
	}

	next := current.Clone()
	if transition.reduce != nil {
		if err := transition.reduce(&next, event); err != nil {
			return Task{}, fmt.Errorf("apply %s: %w", event.eventKind(), err)
		}
	}
	destination, err := transition.destination(next, event)
	if err != nil {
		return Task{}, err
	}
	next.State = destination
	if destination != StateBlocked {
		next.Blocked = nil
	}
	if err := Validate(next); err != nil {
		return Task{}, fmt.Errorf("apply %s: resulting task: %w", event.eventKind(), err)
	}
	return next, nil
}

// Accepts reports whether kind is valid in current's state without applying
// an event. Shells use it before performing I/O needed to construct an event.
func Accepts(current Task, kind EventKind) error {
	state, ok := workflow.states[current.State]
	if !ok {
		return fmt.Errorf("accept event: %w: %q", ErrUnknownState, current.State)
	}
	if state.terminal {
		return newInvalidTransition(current.State, kind)
	}
	if _, ok := workflow.global[kind]; ok {
		return nil
	}
	if _, ok := state.on[kind]; !ok {
		return newInvalidTransition(current.State, kind)
	}
	return nil
}

func (t transition) destination(current Task, event Event) (State, error) {
	if t.to != "" {
		return t.to, nil
	}
	for _, candidate := range t.routes {
		if candidate.when == nil || candidate.when(current, event) {
			return candidate.to, nil
		}
	}
	return "", fmt.Errorf("%w: state=%s event=%s", ErrNoRoute, current.State, event.eventKind())
}

func recordSpecification(t *Task, event Event) error {
	reported, ok := event.(SpecificationSubmitted)
	if !ok {
		return ErrInvalidEventPayload
	}
	if reported.Specification == "" || reported.DoneWhen == "" {
		return ErrInvalidEventPayload
	}
	t.Specification = reported.Specification
	t.DoneWhen = reported.DoneWhen
	return nil
}

func recordSpecificationApproval(t *Task, event Event) error {
	reported, ok := event.(SpecificationApproved)
	if !ok {
		return ErrInvalidEventPayload
	}
	t.SpecificationReviews = append(t.SpecificationReviews, SpecificationReview{
		Approved: true, Comment: reported.Comment, At: reported.At,
	})
	return nil
}

func recordSpecificationRejection(t *Task, event Event) error {
	reported, ok := event.(SpecificationRejected)
	if !ok || reported.Reason == "" {
		return ErrInvalidEventPayload
	}
	t.SpecificationReviews = append(t.SpecificationReviews, SpecificationReview{
		Approved: false, Comment: reported.Reason, At: reported.At,
	})
	return nil
}

func recordVerification(t *Task, event Event) error {
	reported, ok := event.(VerificationReported)
	if !ok || len(reported.Verification.Checks) == 0 {
		return ErrInvalidEventPayload
	}
	reported.Verification.Checks = maps.Clone(reported.Verification.Checks)
	t.Verifications = append(t.Verifications, reported.Verification)
	return nil
}

func recordAutomatedReview(t *Task, event Event) error {
	reported, ok := event.(AutomatedReviewRecorded)
	if !ok {
		return ErrInvalidEventPayload
	}
	t.Reviews = append(t.Reviews, Review{
		Attempt: len(t.Reviews) + 1, Approved: reported.Approved,
		Findings: append([]Finding(nil), reported.Findings...), CreatedAt: reported.At,
	})
	if !reported.Approved && len(t.Reviews) >= 2 {
		t.Blocked = &Blocked{
			ResumeState: StateFixAutomatedReviewFindings,
			Stage:       StageVerification,
			Reason:      fmt.Sprintf("Second review rejected: %s", SummarizeFindings(reported.Findings)),
			At:          reported.At,
		}
	}
	return nil
}

func recordCommit(t *Task, event Event) error {
	reported, ok := event.(CommitRecorded)
	if !ok || reported.Commit.Hash == "" {
		return ErrInvalidEventPayload
	}
	t.Git.Commit = new(reported.Commit)
	return nil
}

func recordHumanApproval(t *Task, event Event) error {
	reported, ok := event.(HumanReviewApproved)
	if !ok {
		return ErrInvalidEventPayload
	}
	t.HumanReviews = append(t.HumanReviews, HumanReview{Approved: true, Comment: reported.Comment, At: reported.At})
	return nil
}

func recordHumanRejection(t *Task, event Event) error {
	reported, ok := event.(HumanReviewRejected)
	if !ok || reported.Reason == "" {
		return ErrInvalidEventPayload
	}
	t.HumanReviews = append(t.HumanReviews, HumanReview{Approved: false, Comment: reported.Reason, At: reported.At})
	return nil
}

func recordMerge(t *Task, event Event) error {
	reported, ok := event.(MergeCompleted)
	if !ok {
		return ErrInvalidEventPayload
	}
	if reported.CommitOverride != "" && t.Git.Commit != nil {
		t.Git.Commit.Hash = reported.CommitOverride
	}
	return nil
}

func recordEscalation(t *Task, event Event) error {
	reported, ok := event.(Escalated)
	if !ok || reported.Stage == "" || reported.Reason == "" {
		return ErrInvalidEventPayload
	}
	t.Blocked = &Blocked{ResumeState: t.State, Stage: reported.Stage, Reason: reported.Reason, At: reported.At}
	return nil
}

func recordAbandonment(t *Task, event Event) error {
	reported, ok := event.(Abandoned)
	if !ok || reported.Reason == "" {
		return ErrInvalidEventPayload
	}
	t.FailureReason = reported.Reason
	return nil
}

func verificationPassed(_ Task, event Event) bool {
	reported, ok := event.(VerificationReported)
	return ok && reported.Verification.Passed()
}

func automatedReviewApproved(_ Task, event Event) bool {
	reported, ok := event.(AutomatedReviewRecorded)
	return ok && reported.Approved
}

func automatedReviewLimitReached(t Task, _ Event) bool { return len(t.Reviews) >= 2 }
func autoApprovedOnTrunk(t Task, _ Event) bool         { return t.AutoApprove && t.Git.Trunk }
func autoApproved(t Task, _ Event) bool                { return t.AutoApprove }
func worksOnTrunk(t Task, _ Event) bool                { return t.Git.Trunk }
func taskIsNotBlocked(t Task, _ Event) bool            { return t.State != StateBlocked }
