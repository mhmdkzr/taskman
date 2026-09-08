// Package approve owns the "review approve" command: its domain logic and
// CLI wiring.
package approve

import (
	"fmt"

	"github.com/mhmdkzr/loop/internal/task"
)

// ApproveReview records a human's approval at the review stage.
func ApproveReview(tasksDir, id string, comment string) (task.Task, error) {
	t, err := task.MutateTask(tasksDir, id, func(t *task.Task) error {
		if t.Status.Review.State != task.StagePending {
			return task.NotInState("review", string(t.Status.Review.State), "pending")
		}
		t.Status.Review.State = task.StageDone
		t.Status.Review.CompletedAt = new(task.Now())
		t.HumanReviews = append(t.HumanReviews, task.HumanReview{Approved: true, Comment: comment, At: task.Now()})
		return nil
	})
	if err != nil {
		return task.Task{}, fmt.Errorf("approve review: %w", err)
	}
	return t, nil
}
