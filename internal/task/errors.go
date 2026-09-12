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
)

func (t Task) invalidTransition(event TaskEvent) error {
	return fmt.Errorf("%w: state=%s event=%s", errInvalidTransition, t.State(), event.Kind())
}
