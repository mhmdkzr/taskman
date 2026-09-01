package codebase

import (
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/mhmdkzr/taskman/internal/semver"
)

func (r Repository) Version() (semver.Version, bool, error) {
	head, err := r.r.Head()
	if err != nil {
		return semver.Version{}, false, err
	}

	tags, err := r.r.Tags()
	if err != nil {
		return semver.Version{}, false, err
	}

	var found string
	tags.ForEach(func(ref *plumbing.Reference) error {
		obj, err := r.r.TagObject(ref.Hash())
		if err != nil {
			return nil // not annotated, skip
		}
		if obj.Target == head.Hash() {
			found = ref.Name().Short()
		}
		return nil
	})

	if found == "" {
		return semver.Version{}, false, nil
	}

	v, err := semver.Parse(found)
	if err != nil {
		return semver.Version{}, false, err
	}

	return v, true, nil
}
