// Package record owns the "review record" command: its domain logic and
// CLI wiring.
package record

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

// Request is task review record's input - one automated review round's
// verdict.
type Request struct {
	Approved bool
	Findings []task.Finding
}

// RecordReview reports the automated review round's verdict inside
// verification - design.md §6's verification stage. Approval advances the
// task to the review stage; rejection either loops back for another
// attempt (attempt 1) or blocks the task (attempt 2, the fixed two-round
// cap).
func RecordReview(tasksDir, id string, req Request) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
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
