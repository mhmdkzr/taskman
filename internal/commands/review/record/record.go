// Package record owns the "review record" command: its domain logic and
// CLI wiring.
package record

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is review record's input - one automated review round's verdict.
// Shared verbatim by the CLI (cmd.go builds it from flags) and MCP (mcp.go
// uses it as the tool's input type directly) frontends; the json/jsonschema
// tags describe it to MCP clients.
type Request struct {
	ID       string         `json:"id"                 jsonschema:"the task id being reviewed"`
	Approved bool           `json:"approved"           jsonschema:"whether the automated review approved this attempt"`
	Findings []task.Finding `json:"findings,omitempty" jsonschema:"findings against the review, each with a file and its full detail text"`
}

func (r Request) validate() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// RecordReview reports the automated review round's verdict inside
// verification - design.md §6's verification stage. Approval advances the
// task to the review stage; rejection either loops back for another
// attempt (attempt 1) or blocks the task (attempt 2, the fixed two-round
// cap).
func RecordReview(tasksDir string, req Request) (task.Task, error) {
	if err := req.validate(); err != nil {
		return task.Task{}, fmt.Errorf("record review: %w", err)
	}
	t, err := task.MutateTask(tasksDir, req.ID, func(t *task.Task) error {
		if t.Status.Implementation.State != task.StageDone {
			return task.NotInState("implementation", string(t.Status.Implementation.State), "done")
		}
		if err := task.NotBlockedOrFailed(t); err != nil {
			return fmt.Errorf("blocked/failed precondition: %w", err)
		}

		attempt := len(t.Reviews) + 1
		t.Reviews = append(t.Reviews, task.Review{
			Attempt:   attempt,
			Approved:  req.Approved,
			Findings:  req.Findings,
			CreatedAt: task.Now(),
		})
		t.Status.Verification.Attempts = attempt

		if req.Approved {
			t.Status.Verification.State = task.StageDone
			t.Status.Verification.CompletedAt = new(task.Now())
			t.Status.Review.State = task.StagePending
			return nil
		}

		if attempt >= 2 {
			t.State = task.StateBlocked
			t.Blocked = &task.Blocked{
				Stage:  task.StageVerification,
				Reason: fmt.Sprintf("Second review rejected: %s", task.SummarizeFindings(req.Findings)),
				At:     task.Now(),
			}
		}
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("record review: %w", err)
	}
	return t, nil
}
