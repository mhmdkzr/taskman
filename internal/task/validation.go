package task

import "fmt"

func mustBuild(candidate definition) definition {
	if err := validateDefinition(candidate); err != nil {
		panic(err)
	}
	return candidate
}

func validateDefinition(candidate definition) error {
	if _, ok := candidate.states[candidate.initial]; !ok {
		return fmt.Errorf("workflow initial state %q is not defined", candidate.initial)
	}
	for stateName, state := range candidate.states {
		if !validInstructionKind(state.instruction.Kind) ||
			!validInstructionAction(state.instruction.Action) {
			return fmt.Errorf("workflow state %q has an invalid instruction", stateName)
		}
		if state.terminal && len(state.on) != 0 {
			return fmt.Errorf("workflow terminal state %q has transitions", stateName)
		}
		for event, transition := range state.on {
			if err := validateTransition(candidate, stateName, event, transition); err != nil {
				return err
			}
		}
	}
	for event, transition := range candidate.global {
		if err := validateTransition(candidate, "<global>", event, transition); err != nil {
			return err
		}
	}
	return nil
}

func validInstructionKind(kind InstructionKind) bool {
	switch kind {
	case InstructionSpecify,
		InstructionSpecificationReview,
		InstructionImplement,
		InstructionVerify,
		InstructionFixVerificationFailure,
		InstructionFixAutomatedReviewFindings,
		InstructionAutomatedReview,
		InstructionCommit,
		InstructionHumanReview,
		InstructionFixHumanReviewFindings,
		InstructionMerge,
		InstructionBlocked,
		InstructionCompleted,
		InstructionAbandoned:
		return true
	default:
		return false
	}
}

func validInstructionAction(action InstructionAction) bool {
	switch action {
	case InstructionDispatch, InstructionRun, InstructionWait, InstructionDone:
		return true
	default:
		return false
	}
}

func validateTransition(
	candidate definition,
	from State,
	event EventKind,
	transition transition,
) error {
	if transition.to != "" && len(transition.routes) != 0 {
		return fmt.Errorf("workflow transition %s/%s has both to and routes", from, event)
	}
	if transition.to == "" && len(transition.routes) == 0 {
		return fmt.Errorf("workflow transition %s/%s has no destination", from, event)
	}
	if transition.to != "" {
		if _, ok := candidate.states[transition.to]; !ok {
			return fmt.Errorf(
				"workflow transition %s/%s has unknown destination %q",
				from,
				event,
				transition.to,
			)
		}
		return nil
	}
	for i, route := range transition.routes {
		if _, ok := candidate.states[route.to]; !ok {
			return fmt.Errorf(
				"workflow transition %s/%s has unknown route destination %q",
				from,
				event,
				route.to,
			)
		}
		if route.when == nil && i != len(transition.routes)-1 {
			return fmt.Errorf(
				"workflow transition %s/%s has a non-final default route",
				from,
				event,
			)
		}
	}
	if transition.routes[len(transition.routes)-1].when != nil {
		return fmt.Errorf("workflow transition %s/%s has no default route", from, event)
	}
	return nil
}

// Validate checks invariants expected by the compiled workflow.
func Validate(t Task) error {
	if t.ID == "" {
		return fmt.Errorf("id is required")
	}
	if t.Definition == "" {
		return fmt.Errorf("definition is required")
	}
	if _, ok := workflow.states[t.State]; !ok {
		return fmt.Errorf("%w: %q", ErrUnknownState, t.State)
	}
	if t.State == StateBlocked && t.Blocked == nil {
		return fmt.Errorf("blocked state requires blocked details")
	}
	if t.State == StateBlocked && t.Blocked != nil {
		resume, ok := workflow.states[t.Blocked.ResumeState]
		if !ok || t.Blocked.ResumeState == StateBlocked || resume.terminal {
			return fmt.Errorf("blocked task has invalid resume state %q", t.Blocked.ResumeState)
		}
	}
	if t.State != StateBlocked && t.Blocked != nil {
		return fmt.Errorf("state %q cannot carry blocked details", t.State)
	}
	return nil
}
