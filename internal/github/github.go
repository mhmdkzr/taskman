package github

import "net/http"

import gh "github.com/google/go-github/v68/github"

// NewClient returns a GitHub API client using the supplied HTTP client.
// Passing nil uses go-github's default HTTP client.
func NewClient(httpClient *http.Client) *gh.Client {
	return gh.NewClient(httpClient)
}
