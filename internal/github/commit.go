package github

import (
	"context"

	"github.com/google/go-github/v68/github"
)

// GetCommit fetches a single commit by SHA or branch name.
func GetCommit(ctx context.Context, c *github.Client, owner, repo, sha string) (*github.RepositoryCommit, error) {
	commit, _, err := c.Repositories.GetCommit(ctx, owner, repo, sha, nil)
	return commit, err
}

// ListCommits returns commits for a repository. opts may be nil for defaults.
func ListCommits(ctx context.Context, c *github.Client, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, error) {
	commits, _, err := c.Repositories.ListCommits(ctx, owner, repo, opts)
	return commits, err
}

// CompareCommits compares two commit SHAs (or branch names) and returns the
// difference between base and head.
func CompareCommits(ctx context.Context, c *github.Client, owner, repo, base, head string) (*github.CommitsComparison, error) {
	comparison, _, err := c.Repositories.CompareCommits(ctx, owner, repo, base, head, nil)
	return comparison, err
}
