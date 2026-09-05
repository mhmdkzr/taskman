// Package glob implements recursive file pattern matching.
package glob

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultLimit = 200

var errPatternRequired = errors.New("pattern is required")

type match struct {
	path    string
	modTime time.Time
}

type output struct {
	Paths     []string `json:"paths"`
	Truncated bool     `json:"truncated"`
}

func execute(_ context.Context, in input) (output, error) {
	root := in.Path
	if root == "" {
		root = "."
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	var matches []match
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %q: %w", path, err)
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relative path %q: %w", path, err)
		}
		if !matchGlob(in.Pattern, filepath.ToSlash(rel)) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("file info %q: %w", path, err)
		}
		matches = append(matches, match{path: path, modTime: info.ModTime()})
		return nil
	})
	if err != nil {
		return output{}, fmt.Errorf("glob: %w", err)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	truncated := len(matches) > limit
	if truncated {
		matches = matches[:limit]
	}

	paths := make([]string, len(matches))
	for i, m := range matches {
		paths[i] = m.path
	}
	return output{Paths: paths, Truncated: truncated}, nil
}

// matchGlob matches a slash-separated relative path against a glob pattern,
// supporting "**" segments that match zero or more path segments.
func matchGlob(pattern, path string) bool {
	return matchGlobSegments(strings.Split(pattern, "/"), strings.Split(path, "/"))
}

func matchGlobSegments(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}

	if pattern[0] == "**" {
		for i := 0; i <= len(path); i++ {
			if matchGlobSegments(pattern[1:], path[i:]) {
				return true
			}
		}
		return false
	}

	if len(path) == 0 {
		return false
	}

	matched, err := filepath.Match(pattern[0], path[0])
	if err != nil || !matched {
		return false
	}
	return matchGlobSegments(pattern[1:], path[1:])
}
