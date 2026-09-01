package codebase

import (
	"bufio"
	"fmt"
	"os"
)

const (
	defaultReadLimit = 2000
	maxScannerBuffer = 1024 * 1024
)

// Read returns a file's content, one line per element, starting at offset
// (0-based) and returning at most limit lines. limit <= 0 uses
// defaultReadLimit. path is resolved against the worktree root and cannot
// escape it (see resolvePath).
func (r Repository) Read(path string, offset, limit int) (lines []string, truncated bool, err error) {
	if offset < 0 {
		return nil, false, fmt.Errorf("read: offset must be >= 0")
	}
	if limit <= 0 {
		limit = defaultReadLimit
	}

	full, err := r.resolvePath(path)
	if err != nil {
		return nil, false, fmt.Errorf("read: %w", err)
	}

	all, err := readLines(full)
	if err != nil {
		return nil, false, fmt.Errorf("read: %w", err)
	}

	if offset > len(all) {
		offset = len(all)
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], end < len(all), nil
}

// readLines reads a file into its lines. A trailing newline does not produce
// a spurious final empty line.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maxScannerBuffer)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
