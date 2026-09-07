package pipeline

import (
	git "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v68/github"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
)

type ReviewResults struct {
	Approved bool
}

func (r ReviewResults) String() string {
	panic("todo")
}

func Review(r *git.Repository, c *github.Client, i *github.Issue, t task.Task) (ReviewResults, error) {
	panic("todo")
}
