// Package github provides GitHub API client helpers.
package github

import (
	"net/http"

	"github.com/google/go-github/v68/github"
)

// NewClient returns a GitHub API client using the supplied HTTP client.
// Passing nil uses go-github's default HTTP client.
func NewClient(httpClient *http.Client) *github.Client {
	return github.NewClient(httpClient)
}
