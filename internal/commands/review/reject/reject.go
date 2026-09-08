// Package reject owns the "review reject" command: its domain logic and
// CLI wiring.
package reject

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is review reject's input. Shared verbatim by the CLI (cmd.go
// builds it from flags) and MCP (mcp.go uses it as the tool's input type
// directly) frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID     string `json:"id"     jsonschema:"the task id being rejected"`
	Reason string `json:"reason" jsonschema:"why the review was rejected"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if r.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// RejectReview records a human's rejection and starts review-reject
// recovery - design.md §6.
func RejectReview(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("reject review: %w", err)
	}
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if t.Status.Review.State != task.StagePending {
			return task.NotInState("review", string(t.Status.Review.State), "pending")
		}
		t.HumanReviews = append(
			t.HumanReviews, task.HumanReview{Approved: false, Comment: req.Reason, At: task.Now()},
		)
		t.Status.Review.State = task.StageInProgress
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("reject review: %w", err)
	}
	return t, nil
}
