package semver

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

type Version struct {
	raw string
}

// BumpType indicates which segment to increment.
type BumpType int

const (
	BumpMajor BumpType = iota
	BumpMinor
	BumpPatch
)

func Parse(s string) (Version, error) {
	if !semver.IsValid(s) {
		return Version{}, &InvalidVersionError{s}
	}
	return Version{raw: s}, nil
}

func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

type InvalidVersionError struct {
	Version string
}

func (e *InvalidVersionError) Error() string {
	return "invalid semantic version: " + e.Version
}

func (v Version) String() string {
	return v.raw
}

func (v Version) IsValid() bool {
	return semver.IsValid(v.raw)
}

func (v Version) Canonical() Version {
	return Version{raw: semver.Canonical(v.raw)}
}

func (v Version) Major() string {
	return semver.Major(v.raw)
}

func (v Version) MajorMinor() string {
	return semver.MajorMinor(v.raw)
}

func (v Version) Prerelease() string {
	return semver.Prerelease(v.raw)
}

func (v Version) Build() string {
	return semver.Build(v.raw)
}

func (v Version) Compare(w Version) int {
	return semver.Compare(v.raw, w.raw)
}

func (v Version) LessThan(w Version) bool {
	return semver.Compare(v.raw, w.raw) < 0
}

func (v Version) GreaterThan(w Version) bool {
	return semver.Compare(v.raw, w.raw) > 0
}

func (v Version) Equal(w Version) bool {
	return semver.Compare(v.raw, w.raw) == 0
}

func IsValid(s string) bool {
	return semver.IsValid(s)
}

func Canonical(s string) string {
	return semver.Canonical(s)
}

func Compare(v, w string) int {
	return semver.Compare(v, w)
}

func Sort(list []string) {
	semver.Sort(list)
}

// numericPart returns the numeric value of a semver segment like "1" or "v1".
func numericPart(s string) (int, error) {
	s = strings.TrimLeft(s, "v")
	return strconv.Atoi(s)
}

func (v Version) numericMajor() (int, error) {
	return numericPart(v.Major())
}

func (v Version) numericMinor() (int, error) {
	mm := v.MajorMinor()
	if _, after, ok := strings.Cut(mm, "."); ok {
		return numericPart(after)
	}
	return 0, fmt.Errorf("cannot parse minor from %q", mm)
}

func (v Version) numericPatch() (int, error) {
	return numericPart(v.raw[strings.LastIndexByte(v.raw, '.')+1:])
}

func bumpRaw(raw string, fn func(major, minor, patch int) (int, int, int)) (string, error) {
	v := Version{raw: raw}
	maj, err := v.numericMajor()
	if err != nil {
		return "", err
	}
	min, err := v.numericMinor()
	if err != nil {
		return "", err
	}
	patch, err := v.numericPatch()
	if err != nil {
		return "", err
	}

	maj, min, patch = fn(maj, min, patch)

	r := fmt.Sprintf("v%d.%d.%d", maj, min, patch)
	if !semver.IsValid(r) {
		return "", &InvalidVersionError{r}
	}
	return r, nil
}

func (v Version) IncMajor() (Version, error) {
	raw, err := bumpRaw(v.raw, func(maj, min, patch int) (int, int, int) {
		return maj + 1, 0, 0
	})
	if err != nil {
		return Version{}, err
	}
	return Version{raw: raw}, nil
}

func (v Version) IncMinor() (Version, error) {
	raw, err := bumpRaw(v.raw, func(maj, min, patch int) (int, int, int) {
		return maj, min + 1, 0
	})
	if err != nil {
		return Version{}, err
	}
	return Version{raw: raw}, nil
}

func (v Version) IncPatch() (Version, error) {
	raw, err := bumpRaw(v.raw, func(maj, min, patch int) (int, int, int) {
		return maj, min, patch + 1
	})
	if err != nil {
		return Version{}, err
	}
	return Version{raw: raw}, nil
}
