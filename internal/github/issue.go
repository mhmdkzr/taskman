package github

import (
	"context"

	"github.com/google/go-github/v68/github"
)

// GetIssue fetches a single issue by number.
func GetIssue(ctx context.Context, c *github.Client, owner, repo string, number int) (*github.Issue, error) {
	issue, _, err := c.Issues.Get(ctx, owner, repo, number)
	return issue, err
}

// ListIssues returns issues for a repository. opts may be nil for defaults.
// Note: the GitHub issues API also returns pull requests; callers can skip
// entries where issue.IsPullRequest() is true.
func ListIssues(ctx context.Context, c *github.Client, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, error) {
	issues, _, err := c.Issues.ListByRepo(ctx, owner, repo, opts)
	return issues, err
}

// CreateIssue opens a new issue. req must have a non-nil Title.
func CreateIssue(ctx context.Context, c *github.Client, owner, repo string, req *github.IssueRequest) (*github.Issue, error) {
	issue, _, err := c.Issues.Create(ctx, owner, repo, req)
	return issue, err
}
