package codebase

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const defaultGrepLimit = 100

// GrepMatch is one matching line.
type GrepMatch struct {
	Path string // relative to the searched path
	Line int
	Text string
}

// Grep searches files under path (relative to the worktree root, default
// ".") for lines matching pattern. include, if set, is a glob (** supported)
// matching file paths must satisfy. limit <= 0 uses defaultGrepLimit.
func (r Repository) Grep(pattern, path, include string, limit int) (matches []GrepMatch, truncated bool, err error) {
	if pattern == "" {
		return nil, false, fmt.Errorf("grep: pattern is required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, false, fmt.Errorf("grep: invalid pattern: %w", err)
	}
	if path == "" {
		path = "."
	}
	if limit <= 0 {
		limit = defaultGrepLimit
	}
	root, err := r.resolvePath(path)
	if err != nil {
		return nil, false, fmt.Errorf("grep: %w", err)
	}

	var includeRe *regexp.Regexp
	if include != "" {
		includeRe = globToRegex(include)
	}

	werr := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if len(matches) >= limit {
			truncated = true
			return filepath.SkipAll
		}
		if includeRe != nil {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return nil
			}
			if !includeRe.MatchString(filepath.ToSlash(rel)) {
				return nil
			}
		}
		fileMatches, err := grepFile(root, p, re, limit-len(matches))
		if err != nil {
			return nil
		}
		matches = append(matches, fileMatches...)
		if len(matches) >= limit {
			truncated = true
		}
		return nil
	})
	if werr != nil && !errors.Is(werr, filepath.SkipAll) {
		return nil, false, fmt.Errorf("grep: %w", werr)
	}
	return matches, truncated, nil
}

func grepFile(root, path string, re *regexp.Regexp, max int) ([]GrepMatch, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	if rel == "." {
		rel = filepath.Base(path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []GrepMatch
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maxScannerBuffer)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if re.MatchString(scanner.Text()) {
			out = append(out, GrepMatch{Path: filepath.ToSlash(rel), Line: lineNum, Text: strings.TrimSpace(scanner.Text())})
			if len(out) >= max {
				break
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
