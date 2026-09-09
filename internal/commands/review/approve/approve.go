// Package approve owns the "review approve" command: its domain logic and
// CLI wiring.
package approve

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is review approve's input. Shared verbatim by the CLI (cmd.go
// builds it from flags) and MCP (mcp.go uses it as the tool's input type
// directly) frontends; the json/jsonschema tags describe it to MCP clients.
type Request struct {
	ID      string `json:"id"                jsonschema:"the task id being approved"`
	Comment string `json:"comment,omitempty" jsonschema:"an optional note, e.g. LGTM"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// ApproveReview records a human's approval at the review stage.
func ApproveReview(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("approve review: %w", err)
	}
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if t.Status.Review.State != task.StagePending {
			return task.NotInState("review", string(t.Status.Review.State), "pending")
		}
		t.Status.Review.State = task.StageDone
		t.Status.Review.CompletedAt = new(task.Now())
		t.HumanReviews = append(
			t.HumanReviews, task.HumanReview{Approved: true, Comment: req.Comment, At: task.Now()},
		)
		task.CompleteTrunkMerge(t)
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("approve review: %w", err)
	}
	return t, nil
}
