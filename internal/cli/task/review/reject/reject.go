// Package reject owns the "review reject" command: its domain logic and
// CLI wiring.
package reject

import (
	"fmt"

	"github.com/mhmdkzr/loop/internal/task"
)

// RejectReview records a human's rejection and starts review-reject
// recovery - design.md §6.
func RejectReview(tasksDir, id string, reason string) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.Status.Review.State != task.StagePending {
			return task.NotInState("review", string(t.Status.Review.State), "pending")
		}
		t.HumanReviews = append(t.HumanReviews, task.HumanReview{Approved: false, Comment: reason, At: task.Now()})
		t.Status.Review.State = task.StageInProgress
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("reject review: %w", err)
	}
	return t, nil
}
