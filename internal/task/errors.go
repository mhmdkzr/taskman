package task

import (
	"errors"
	"strings"
)

var (
	// ErrTaskNotFound is returned when a task id has no corresponding file.
	ErrTaskNotFound = errors.New("task not found")
	// ErrWorkingTreeDirty is returned by Create when the working tree isn't
	// clean.
	ErrWorkingTreeDirty = errors.New("working tree is not clean")
	// ErrInvalidLabel is returned when a well-known label key is set to a
	// value outside its fixed enum.
	ErrInvalidLabel = errors.New("invalid label value")
	// ErrUnknownState is returned when a task or definition names a state
	// absent from the compiled workflow.
	ErrUnknownState = errors.New("unknown workflow state")
	// ErrNoRoute is returned when a routed transition has no matching route.
	ErrNoRoute = errors.New("no transition route")
	// ErrInvalidEventPayload is returned when an event's typed data is invalid.
	ErrInvalidEventPayload = errors.New("invalid event payload")
)

// InvalidTransitionError is returned when a command's precondition isn't
// met.
type InvalidTransitionError struct {
	State State
	Event EventKind
}

func (e *InvalidTransitionError) Error() string {
	return "invalid transition: state=" + string(e.State) + " event=" + string(e.Event)
}

func newInvalidTransition(state State, event EventKind) error {
	return &InvalidTransitionError{State: state, Event: event}
}

// SummarizeFindings joins findings into one line, for use in a Blocked
// reason or a fix prompt.
func SummarizeFindings(findings []Finding) string {
	if len(findings) == 0 {
		return "no findings reported"
	}
	parts := make([]string, len(findings))
	for i, f := range findings {
		parts[i] = f.File + ": " + f.Detail
	}
	return strings.Join(parts, "; ")
}
