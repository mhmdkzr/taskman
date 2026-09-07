package pipeline

import (
	git "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v68/github"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
)

type CommitMessage struct {
	Type  string
	Title string
	Body  string
}

func (c *CommitMessage) String() string {
	panic("todo")
}

func CreateCommitMessage(r *git.Repository, i *github.Issue, t task.Task) (CommitMessage, error) {
	panic("todo")
}
