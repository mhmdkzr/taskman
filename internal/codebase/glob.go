package codebase

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Glob finds files under path (relative to the worktree root, default ".")
// matching pattern. ** matches across directories, * matches within a
// directory, ? matches a single character. The repository's own .git
// directory is skipped; other dotfiles/dot-directories (e.g. .golangci.yaml)
// are walked and matched like any other path. Returned paths are relative to
// path.
func (r Repository) Glob(pattern, path string) ([]string, error) {
	if pattern == "" {
		return nil, fmt.Errorf("glob: pattern is required")
	}
	if path == "" {
		path = "."
	}
	root, err := r.resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("glob: %w", err)
	}

	re := globToRegex(pattern)

	var matches []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		if re.MatchString(filepath.ToSlash(rel)) {
			matches = append(matches, rel)
		}
		return nil
	})
	return matches, nil
}

// globToRegex converts a glob pattern to an anchored regular expression.
// Supported syntax: ** (across directories), * (within a directory), and ?
// (a single non-separator character). All other characters match literally.
// The generated pattern is always valid, so no error is returned.
func globToRegex(pattern string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				for i+1 < len(pattern) && pattern[i+1] == '*' {
					i++
				}
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}
