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
