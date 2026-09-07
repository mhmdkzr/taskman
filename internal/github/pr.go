package github

import (
	"context"

	"github.com/google/go-github/v68/github"
)

// GetPullRequest fetches a single pull request by number.
func GetPullRequest(ctx context.Context, c *github.Client, owner, repo string, number int) (*github.PullRequest, error) {
	pr, _, err := c.PullRequests.Get(ctx, owner, repo, number)
	return pr, err
}

// ListPullRequests returns pull requests for a repository. opts may be nil for defaults.
func ListPullRequests(ctx context.Context, c *github.Client, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, error) {
	prs, _, err := c.PullRequests.List(ctx, owner, repo, opts)
	return prs, err
}

// CreatePullRequest opens a new pull request. pull must have non-nil Title,
// Head, and Base.
func CreatePullRequest(ctx context.Context, c *github.Client, owner, repo string, pull *github.NewPullRequest) (*github.PullRequest, error) {
	pr, _, err := c.PullRequests.Create(ctx, owner, repo, pull)
	return pr, err
}
