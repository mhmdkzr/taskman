package codebase

import (
	"errors"
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// FetchBranch fetches branch from the local repository at fromPath into this
// repository, creating or updating a local branch of the same name and
// copying the objects it needs along with it. It does not check out or
// merge anything — the fetched commit is simply made reachable in this
// repository's own object store for a caller to inspect or merge itself.
//
// This exists for Clone's counterpart: a task's work happens in an isolated
// clone so nothing it does can touch the source repository directly: once
// it produces a commit worth keeping, that commit only really exists once
// it's fetched back here — the clone is disposable and typically removed
// once its work is done.
func (r Repository) FetchBranch(fromPath, branch string) error {
	remote, err := r.r.CreateRemoteAnonymous(&config.RemoteConfig{
		Name: "anonymous",
		URLs: []string{fromPath},
	})
	if err != nil {
		return fmt.Errorf("create anonymous remote: %w", err)
	}

	ref := plumbing.NewBranchReferenceName(branch)
	refSpec := config.RefSpec(fmt.Sprintf("%s:%s", ref, ref))
	err = remote.Fetch(&git.FetchOptions{RefSpecs: []config.RefSpec{refSpec}})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch %s from %s: %w", branch, fromPath, err)
	}
	return nil
}
