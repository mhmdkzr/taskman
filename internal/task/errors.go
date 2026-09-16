package task

import (
	"errors"
	"fmt"
)

var (
	errAlreadyPresent      = errors.New("task part is already present")
	errMissingTaskData     = errors.New("required task data is missing")
	errInvalidTransition   = errors.New("invalid task state transition")
	errInvalidEventPayload = errors.New("event payload is invalid for its kind")
	errUnknownState        = errors.New("unknown task state")
	errNoRoute             = errors.New("no matching route for state and event")

	// ErrUnknownEventKind is returned by DecodeEvent for a kind with no
	// registered decoder. Exported so a caller decoding a stored EventKind
	// (e.g. internal/task/store) can distinguish "this kind doesn't exist"
	// from an ordinary malformed-payload error.
	ErrUnknownEventKind = errors.New("unknown event kind")
)

func (t Task) invalidTransition(event TaskEvent) error {
	return fmt.Errorf("%w: state=%s event=%s", errInvalidTransition, t.State(), event.Kind())
}
