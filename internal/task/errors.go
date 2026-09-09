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
)

// InvalidTransitionError is returned when a command's precondition isn't
// met.
type InvalidTransitionError struct {
	Stage string
	Have  string
	Want  string
}

func (e *InvalidTransitionError) Error() string {
	return "invalid transition: " + e.Stage + " is " + e.Have + ", want " + e.Want
}

// NotInState builds the error every command's precondition check returns
// when a stage isn't in the state it needs to be in for that command to
// apply.
func NotInState(stage, have, want string) error {
	return &InvalidTransitionError{Stage: stage, Have: have, Want: want}
}

// NotBlockedOrFailed returns an error unless t is neither blocked nor
// failed - the precondition task verify and task review record share.
func NotBlockedOrFailed(t *Task) error {
	if t.State == StateBlocked || t.State == StateFailed {
		return &InvalidTransitionError{Stage: "task", Have: string(t.State), Want: "not blocked/failed"}
	}
	return nil
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
