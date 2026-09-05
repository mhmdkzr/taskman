# GitHub API Skill

Use this skill when working with the GitHub API through `internal/github`.

## Package

- Import `github.com/google/go-github/v68/github`.
- Do not use the legacy unversioned `github.com/google/go-github` path.
- The repository wraps client construction in `internal/github.NewClient`.
- Dependency version: `v68.0.0`.

## Client Construction

`go-github` uses `*github.Client` with service fields for API areas:

```go
client := github.NewClient(httpClient)
repo, resp, err := client.Repositories.Get(ctx, owner, repoName)
```

Passing `nil` to `github.NewClient` uses the default HTTP client. For a token,
use `github.NewTokenClient(ctx, token)` or construct a client and call
`client.WithAuthToken(token)`. Prefer an injected `*http.Client` when custom
transport, timeouts, or test behavior are required.

## Services

Use the service matching the GitHub API resource:

- `client.Repositories` for repositories, contents, branches, releases, and commits.
- `client.Issues` for issues, labels, milestones, and issue comments.
- `client.PullRequests` for pull requests, reviews, files, and PR comments.
- `client.Users` for user operations.
- `client.Search` for search endpoints.

Service methods generally accept `context.Context` and return a resource, a
`*github.Response`, and an `error`.

## Pagination

List methods that use offset pagination accept `*github.ListOptions`:

```go
issues, _, err := client.Issues.ListByRepo(ctx, owner, repoName, &github.IssueListByRepoOptions{
	ListOptions: github.ListOptions{Page: 1, PerPage: 30},
})
```

Check `response.NextPage` when walking pages. Use the endpoint-specific options
type when the method requires one, and `github.ListOptions` for simple lists.

## Errors and Responses

Always return or wrap API errors. Keep the response when callers need status,
headers, pagination, or rate-limit data. Use `errors.As` with `*github.ErrorResponse`
when behavior depends on an HTTP error status.

```go
repo, resp, err := client.Repositories.Get(ctx, owner, repoName)
if err != nil {
	var apiErr *github.ErrorResponse
	if errors.As(err, &apiErr) && apiErr.Response.StatusCode == http.StatusNotFound {
		// Handle an absent repository.
	}
	return nil, fmt.Errorf("get repository: %w", err)
}
_ = resp
return repo, nil
```

## Verification

Read the installed API before using unfamiliar methods:

```bash
go doc github.com/google/go-github/v68/github.Client
go doc github.com/google/go-github/v68/github.RepositoriesService
go doc github.com/google/go-github/v68/github.IssuesService
go doc github.com/google/go-github/v68/github.PullRequestsService
```

Run the smallest relevant package tests, then `go vet` and `go test` for the
affected packages.
