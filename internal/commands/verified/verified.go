// Package verified owns the "verified" command: it reports one
// verification attempt. Whether that attempt passed or failed is derived
// from the reported checks, not a separate input - there is no way to
// report a failing check yet call it a pass.
package verified

import (
	"fmt"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is verified's input.
type Request struct {
	ID     uuid.UUID   `json:"id"               jsonschema:"the task whose verification was reported"`
	Checks task.Checks `json:"checks"           jsonschema:"each required check's outcome"`
	Output string      `json:"output,omitempty" jsonschema:"verification output/log text"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if r.Checks == (task.Checks{}) {
		return fmt.Errorf("at least one check result is required")
	}
	return nil
}

func (r Request) failed() bool {
	return r.Checks.Unit == task.CheckError ||
		r.Checks.Integration == task.CheckError ||
		r.Checks.EndToEnd == task.CheckError ||
		r.Checks.Linters == task.CheckError
}

// Verified reports req's verification attempt and returns the resulting
// task.
func Verified(st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record verification: %w", err)
	}
	at := time.Now().UTC()
	var event task.TaskEvent
	if req.failed() {
		event = task.VerificationFailed{Checks: req.Checks, Output: req.Output, At: at}
	} else {
		event = task.VerificationPassed{Checks: req.Checks, Output: req.Output, At: at}
	}
	t, err := st.Append(req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("record verification: %w", err)
	}
	return t, nil
}
