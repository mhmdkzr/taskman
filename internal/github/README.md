# GitHub

Thin helpers over [go-github](https://github.com/google/go-github) for the
GitHub REST API. It is not a vertical slice: there are no HTTP routes, no
storage, and no events - just package-level functions that wrap go-github
client calls so callers do not deal with the `resp` return value.

## Functionality

- `client.go`: `NewClient` builds a `*github.Client` from an optional
  `*http.Client` (nil uses go-github's default).
- `commit.go`: `GetCommit`, `ListCommits`, and `CompareCommits` for reading
  and comparing commits.
- `issue.go`: `GetIssue`, `ListIssues`, and `CreateIssue` for issues. Note
  that GitHub's issues API also returns pull requests; callers can skip
  entries where `issue.IsPullRequest()` is true.
- `pr.go`: `GetPullRequest`, `ListPullRequests`, and `CreatePullRequest`
  for pull requests.

## Behavior

Each helper maps 1:1 onto a go-github call: it performs the request and
returns `(result, error)`, discarding the `*github.Response`. Options structs
(`*github.CommitsListOptions`, etc.) may be nil for defaults. Create helpers
require the caller to supply a well-formed request (e.g. a non-nil `Title`
for `CreateIssue`; non-nil `Title`, `Head`, and `Base` for
`CreatePullRequest`) - errors from the API are returned unwrapped.

## Invocation

Callers construct a client with `github.NewClient` (typically with an
OAuth-capable `*http.Client`) and pass it to the helpers:

```go
client := github.NewClient(nil)
commit, err := github.GetCommit(ctx, client, "owner", "repo", "main")
```
