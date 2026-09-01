package codebase

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/semver"
)

func (r Repository) Bump(how semver.BumpType) (semver.Version, error) {
	v, found, err := r.Version()
	if err != nil {
		return semver.Version{}, fmt.Errorf("get version: %w", err)
	}
	if !found {
		v = semver.MustParse("v0.0.0")
	}

	var bumped semver.Version
	switch how {
	case semver.BumpMajor:
		bumped, err = v.IncMajor()
	case semver.BumpMinor:
		bumped, err = v.IncMinor()
	case semver.BumpPatch:
		bumped, err = v.IncPatch()
	default:
		return semver.Version{}, fmt.Errorf("unknown bump type: %d", how)
	}
	if err != nil {
		return semver.Version{}, fmt.Errorf("bump %s: %w", v, err)
	}

	return bumped, nil
}
