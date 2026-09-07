package pipeline

import (
	git "github.com/go-git/go-git/v5"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
)

func Implement(r *git.Repository, t task.Task) error {
	panic("todo")
}

func FixIssuesFromReview(r *git.Repository, t task.Task, reviewResults ReviewResults) error {
	panic("todo")
}
