package pipeline

import (
	"github.com/google/go-github/v68/github"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
)

func CreateTask(c *github.Client, i *github.Issue) (task.Task, error) {
	panic("todo")
}

func MarkTaskAsDone(t task.Task, pr *github.PullRequest) error {
	panic("todo")
}

func MarkTaskAsFailed(t task.Task, reason string) error {
	panic("todo")
}

func MarkTaskAsInProgress(t task.Task) error {
	panic("todo")
}

func MarkTaskAsReadyForReview(t task.Task) error {
	panic("todo")
}
