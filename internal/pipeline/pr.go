package pipeline

import (
	git "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v68/github"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
)

func CreatePullRequest(r *git.Repository, c *github.Client, i *github.Issue, t task.Task) (*github.PullRequest, error) {
	panic("todo")
}
