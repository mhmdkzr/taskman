package task

import "fmt"

// Apply evaluates event against current's workflow position and returns the
// resulting task. current is never mutated - on any error the zero Task is
// returned alongside it, so callers never observe a partially applied event.
func Apply(current Task, event TaskEvent) (Task, error) {
	if event == nil {
		return Task{}, fmt.Errorf("apply event: event is required")
	}
	if err := current.Validate(); err != nil {
		return Task{}, fmt.Errorf("apply event: current task: %w", err)
	}

	state, ok := workflow.states[current.State()]
	if !ok {
		return Task{}, fmt.Errorf("apply event: %w: %q", errUnknownState, current.State())
	}
	if state.terminal {
		return Task{}, current.invalidTransition(event)
	}

	tr, ok := workflow.global[event.Kind()]
	if !ok {
		tr, ok = state.on[event.Kind()]
	}
	if !ok {
		return Task{}, current.invalidTransition(event)
	}
	if tr.guard != nil && !tr.guard(current, event) {
		return Task{}, current.invalidTransition(event)
	}

	next := current.Clone()
	if tr.reduce != nil {
		if err := tr.reduce(&next, event); err != nil {
			return Task{}, fmt.Errorf("apply %s: %w", event.Kind(), err)
		}
	}
	destination, err := tr.destination(next, event)
	if err != nil {
		return Task{}, err
	}
	next.StateHistory = append(next.StateHistory, StateChange{State: destination, At: event.OccurredAt()})
	if err := next.Validate(); err != nil {
		return Task{}, fmt.Errorf("apply event: resulting task: %w", err)
	}
	return next, nil
}

func recordSpecification(t *Task, event TaskEvent) error {
	reported, ok := event.(SpecificationSubmitted)
	if !ok {
		return errInvalidEventPayload
	}
	if reported.Specification.Plan == "" {
		return fmt.Errorf("record specification: plan is required")
	}
	t.Specification = &reported.Specification
	return nil
}

func recordSpecificationReviewHumanApproved(t *Task, event TaskEvent) error {
	reported, ok := event.(SpecificationReviewHumanApproved)
	if !ok {
		return errInvalidEventPayload
	}
	t.Specification.Review.Human.Results = append(
		t.Specification.Review.Human.Results,
		HumanReviewResult{Approved: true, Comment: reported.Comment, At: reported.At},
	)
	return nil
}

func recordSpecificationReviewHumanRejected(t *Task, event TaskEvent) error {
	reported, ok := event.(SpecificationReviewHumanRejected)
	if !ok || reported.Reason == "" {
		return errInvalidEventPayload
	}
	t.Specification.Review.Human.Results = append(
		t.Specification.Review.Human.Results,
		HumanReviewResult{Comment: reported.Reason, At: reported.At},
	)
	return nil
}

func recordSpecificationReviewAgentApproved(t *Task, event TaskEvent) error {
	reported, ok := event.(SpecificationReviewAgentApproved)
	if !ok {
		return errInvalidEventPayload
	}
	t.Specification.Review.Agent.Results = append(
		t.Specification.Review.Agent.Results,
		AgentReviewResult{Approved: true, Comment: reported.Comment, At: reported.At},
	)
	return nil
}

func recordSpecificationReviewAgentRejected(t *Task, event TaskEvent) error {
	reported, ok := event.(SpecificationReviewAgentRejected)
	if !ok || len(reported.Findings) == 0 {
		return errInvalidEventPayload
	}
	t.Specification.Review.Agent.Results = append(
		t.Specification.Review.Agent.Results,
		AgentReviewResult{Findings: append([]Finding(nil), reported.Findings...), At: reported.At},
	)
	return nil
}

func recordImplementation(t *Task, event TaskEvent) error {
	reported, ok := event.(ImplementationCompleted)
	if !ok {
		return errInvalidEventPayload
	}
	if t.Implementation != nil {
		return fmt.Errorf("record implementation: %w", errAlreadyPresent)
	}
	t.Implementation = &reported.Implementation
	return nil
}

func recordVerificationPassed(t *Task, event TaskEvent) error {
	reported, ok := event.(VerificationPassed)
	if !ok || reported.Checks == (Checks{}) {
		return errInvalidEventPayload
	}
	t.Implementation.Verification.Attempts = append(
		t.Implementation.Verification.Attempts,
		VerificationResult{Passed: true, Checks: reported.Checks, Output: reported.Output, At: reported.At},
	)
	return nil
}

func recordVerificationFailed(t *Task, event TaskEvent) error {
	reported, ok := event.(VerificationFailed)
	if !ok || reported.Checks == (Checks{}) {
		return errInvalidEventPayload
	}
	t.Implementation.Verification.Attempts = append(
		t.Implementation.Verification.Attempts,
		VerificationResult{Checks: reported.Checks, Output: reported.Output, At: reported.At},
	)
	// A verification failure only spends the verification auto-fix budget at
	// the state that owns that loop (verify and its retry state). A failure
	// inside a review-driven fix state still loops through verification, but
	// its own gate's budget is charged when that review rejects.
	switch t.State() { //nolint:exhaustive // only the verification loop's own states charge its budget.
	case StateVerify, StateFixVerificationFailure:
		markAutoFixBudgetExhausted(t, StageVerification,
			t.Implementation.Verification.AutoFix, t.Implementation.Verification.Unblocks, failedVerificationCount(t))
	}
	return nil
}

func recordImplementationReviewAgentApproved(t *Task, event TaskEvent) error {
	reported, ok := event.(ImplementationReviewAgentApproved)
	if !ok {
		return errInvalidEventPayload
	}
	t.Implementation.Review.Agent.Results = append(
		t.Implementation.Review.Agent.Results,
		AgentReviewResult{Approved: true, Comment: reported.Comment, At: reported.At},
	)
	return nil
}

func recordImplementationReviewAgentRejected(t *Task, event TaskEvent) error {
	reported, ok := event.(ImplementationReviewAgentRejected)
	if !ok || len(reported.Findings) == 0 {
		return errInvalidEventPayload
	}
	t.Implementation.Review.Agent.Results = append(
		t.Implementation.Review.Agent.Results,
		AgentReviewResult{Findings: append([]Finding(nil), reported.Findings...), At: reported.At},
	)
	markAutoFixBudgetExhausted(t, StageAutomatedReview,
		t.Implementation.Review.Agent.AutoFix, t.Implementation.Review.Agent.Unblocks, rejectedAgentReviewCount(t))
	return nil
}

func recordCommit(t *Task, event TaskEvent) error {
	reported, ok := event.(CommitRecorded)
	if !ok || reported.Commit.Hash == "" {
		return errInvalidEventPayload
	}
	t.Implementation.Git.Commits = append(t.Implementation.Git.Commits, reported.Commit)
	return nil
}

func recordImplementationReviewHumanApproved(t *Task, event TaskEvent) error {
	reported, ok := event.(ImplementationReviewHumanApproved)
	if !ok {
		return errInvalidEventPayload
	}
	t.Implementation.Review.Human.Results = append(
		t.Implementation.Review.Human.Results,
		HumanReviewResult{Approved: true, Comment: reported.Comment, At: reported.At},
	)
	return nil
}

func recordImplementationReviewHumanRejected(t *Task, event TaskEvent) error {
	reported, ok := event.(ImplementationReviewHumanRejected)
	if !ok || reported.Reason == "" {
		return errInvalidEventPayload
	}
	t.Implementation.Review.Human.Results = append(
		t.Implementation.Review.Human.Results,
		HumanReviewResult{Comment: reported.Reason, At: reported.At},
	)
	markAutoFixBudgetExhausted(t, StageHumanReview,
		t.Implementation.Review.Human.AutoFix, t.Implementation.Review.Human.Unblocks, rejectedHumanReviewCount(t))
	return nil
}

func recordMerge(t *Task, event TaskEvent) error {
	reported, ok := event.(MergeCompleted)
	if !ok || reported.Merge.Target == "" || reported.Merge.Commit == "" {
		return errInvalidEventPayload
	}
	if len(t.Implementation.Git.Commits) == 0 {
		return fmt.Errorf("record merge: %w: source commit", errMissingTaskData)
	}
	if t.Implementation.Git.Merge != nil {
		return fmt.Errorf("record merge: %w", errAlreadyPresent)
	}
	t.Implementation.Git.Merge = &reported.Merge
	return nil
}

func recordEscalation(t *Task, event TaskEvent) error {
	reported, ok := event.(Escalated)
	if !ok || reported.Stage == "" || reported.Reason == "" {
		return errInvalidEventPayload
	}
	t.Blocked = &Blockage{Stage: reported.Stage, Reason: reported.Reason}
	return nil
}

func recordAbandonment(t *Task, event TaskEvent) error {
	reported, ok := event.(Abandoned)
	if !ok || reported.Reason == "" {
		return errInvalidEventPayload
	}
	t.Blocked = nil
	t.Abandoned = &Abandonment{Reason: reported.Reason, At: reported.At}
	return nil
}

// recordUnblock clears t's Blockage and, for a budget-exhaustion blockage,
// grants Rounds additional auto-fix rounds to the gate named by
// t.Blocked.Stage. Rounds must be positive for a budget-exhaustion blockage
// (a zero grant would just resume and immediately re-block on the next
// failure) and must be absent for an escalation-caused blockage, which has
// no budget to grant against.
func recordUnblock(t *Task, event TaskEvent) error {
	reported, ok := event.(Unblocked)
	if !ok || reported.Reason == "" {
		return errInvalidEventPayload
	}
	switch t.Blocked.Stage {
	case StageVerification:
		if reported.Rounds <= 0 {
			return fmt.Errorf("record unblock: rounds must be greater than zero to resume a verification budget block")
		}
		t.Implementation.Verification.Unblocks = append(t.Implementation.Verification.Unblocks,
			Unblock{Rounds: reported.Rounds, Reason: reported.Reason, At: reported.At})
	case StageAutomatedReview:
		if reported.Rounds <= 0 {
			return fmt.Errorf("record unblock: rounds must be greater than zero to resume an automated review " +
				"budget block")
		}
		t.Implementation.Review.Agent.Unblocks = append(t.Implementation.Review.Agent.Unblocks,
			Unblock{Rounds: reported.Rounds, Reason: reported.Reason, At: reported.At})
	case StageHumanReview:
		if reported.Rounds <= 0 {
			return fmt.Errorf("record unblock: rounds must be greater than zero to resume a human review budget block")
		}
		t.Implementation.Review.Human.Unblocks = append(t.Implementation.Review.Human.Unblocks,
			Unblock{Rounds: reported.Rounds, Reason: reported.Reason, At: reported.At})
	default:
		if reported.Rounds != 0 {
			return fmt.Errorf("record unblock: rounds are not applicable to an escalation-caused blockage")
		}
	}
	t.Blocked = nil
	return nil
}

// recordLabelsUpdated sets reported.Set's keys in t's labels and deletes
// reported.Remove's keys, in that order. At least one of the two must be
// non-empty - a call that changes nothing is refused rather than silently
// accepted.
func recordLabelsUpdated(t *Task, event TaskEvent) error {
	reported, ok := event.(LabelsUpdated)
	if !ok {
		return errInvalidEventPayload
	}
	if len(reported.Set) == 0 && len(reported.Remove) == 0 {
		return fmt.Errorf("record labels updated: at least one label to set or remove is required")
	}
	if len(reported.Set) > 0 && t.Definition.Labels == nil {
		t.Definition.Labels = make(map[string]string, len(reported.Set))
	}
	for k, v := range reported.Set {
		t.Definition.Labels[k] = v
	}
	for _, k := range reported.Remove {
		delete(t.Definition.Labels, k)
	}
	if len(t.Definition.Labels) == 0 {
		t.Definition.Labels = nil
	}
	return nil
}
