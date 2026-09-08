package task

import "errors"

var (
	// ErrTaskNotFound is returned when a task id has no corresponding file.
	ErrTaskNotFound = errors.New("task not found")
	// ErrWorkingTreeDirty is returned by Create when the working tree isn't
	// clean - design.md §5.
	ErrWorkingTreeDirty = errors.New("working tree is not clean")
	// ErrInvalidLabel is returned when a well-known label key is set to a
	// value outside its fixed enum - design.md §3.
	ErrInvalidLabel = errors.New("invalid label value")
)

// InvalidTransitionError is returned when a command's precondition isn't
// met - design.md §6's per-command precondition column.
type InvalidTransitionError struct {
	Stage string
	Have  string
	Want  string
}

func (e *InvalidTransitionError) Error() string {
	return "invalid transition: " + e.Stage + " is " + e.Have + ", want " + e.Want
}
