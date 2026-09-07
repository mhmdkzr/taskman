package pipeline

import (
	git "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v68/github"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
)

func Start(r *git.Repository, c *github.Client) error {
	issues, err := Discover(r)
	if err != nil {
		return err
	}

	for _, i := range issues {
		t, err := CreateTask(c, i)
		if err != nil {
			return err
		}

		if err = ProcessTask(r, c, i, t); err != nil {
			if err := MarkTaskAsFailed(t, err.Error()); err != nil {
				return err
			}
			continue
		}
	}

	return nil
}

func ProcessTask(r *git.Repository, c *github.Client, i *github.Issue, t task.Task) error {
	if err := MarkTaskAsInProgress(t); err != nil {
		return err
	}

	if err := Implement(r, t); err != nil {
		return err
	}

	if err := MarkTaskAsReadyForReview(t); err != nil {
		return err
	}

	reviewResults, err := Review(r, c, i, t)
	if err != nil {
		return err
	}

	if !reviewResults.Approved {
		if err := FixIssuesFromReview(r, t, reviewResults); err != nil {
			return err
		}

		if reviewResults, err = Review(r, c, i, t); err != nil {
			return err
		}

		if !reviewResults.Approved {
			return MarkTaskAsFailed(t, reviewResults.String())
		}
	}

	msg, err := CreateCommitMessage(r, i, t)
	if err != nil {
		return err
	}

	if err := Commit(r, msg); err != nil {
		return err
	}

	pr, err := CreatePullRequest(r, c, i, t)
	if err != nil {
		return err
	}

	return MarkTaskAsDone(t, pr)
}
