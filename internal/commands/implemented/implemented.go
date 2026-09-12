// Package implemented owns the "implemented" command: it reports a task's
// implementation as complete.
package implemented

import (
	"fmt"
	"time"

	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

// Request is implemented's input.
type Request struct {
	ID           uuid.UUID                `json:"id"                     jsonschema:"the task that was implemented"`
	Worktree     string                   `json:"worktree"                jsonschema:"the worktree the implementation was done in"`
	Branch       string                   `json:"branch"                  jsonschema:"the branch the implementation was done on"`
	Verification task.Verification        `json:"verification,omitempty"  jsonschema:"which verification checks this implementation requires"`
	Review       task.ReviewConfiguration `json:"review,omitempty" jsonschema:"which review gates this implementation requires"`
}

func (r Request) validate() error {
	if r.ID == uuid.Nil() {
		return fmt.Errorf("id is required")
	}
	if r.Worktree == "" {
		return fmt.Errorf("worktree is required")
	}
	if r.Branch == "" {
		return fmt.Errorf("branch is required")
	}
	return nil
}

// Implemented reports req's implementation as complete and returns the
// resulting task.
func Implemented(st *store.Store, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record implementation: %w", err)
	}
	event := task.ImplementationCompleted{
		Implementation: task.Implementation{
			Git:          task.Git{Worktree: req.Worktree, Branch: req.Branch},
			Verification: req.Verification,
			Review:       req.Review,
		},
		At: time.Now().UTC(),
	}
	t, err := st.Append(req.ID, event)
	if err != nil {
		return task.Task{}, fmt.Errorf("record implementation: %w", err)
	}
	return t, nil
}
