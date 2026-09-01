package codebase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditResult reports what Edit did.
type EditResult struct {
	// Created is true if the file did not exist before this call.
	Created bool
	// BytesWritten is the length of the content written (create) or added
	// (append), or the length of the whole file after a replace.
	BytesWritten int
}

// Edit is the one write-side file operation: it covers create, append, and
// exact-match replace behind a single interface, chosen by whether old is
// empty and whether the file already exists —
//   - file does not exist, old == ""   -> create it with content newText
//   - file exists,        old == ""    -> append newText to the end
//   - file exists,        old != ""    -> old must match exactly once in the
//     current content; replaced with newText (old matching zero or more than
//     once is an error, so a caller can't silently touch the wrong text),
//     unless replaceAll is set, in which case every occurrence is replaced
//     and old matching zero times is still an error
//
// path is resolved against the worktree root and cannot escape it.
func (r Repository) Edit(path, old, newText string, replaceAll bool) (*EditResult, error) {
	full, err := r.resolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("edit: %w", err)
	}

	existing, err := os.ReadFile(full)
	switch {
	case os.IsNotExist(err):
		if old != "" {
			return nil, fmt.Errorf("edit: %s does not exist; old must be empty to create it", path)
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, fmt.Errorf("edit: %w", err)
		}
		if err := os.WriteFile(full, []byte(newText), 0o644); err != nil {
			return nil, fmt.Errorf("edit: %w", err)
		}
		return &EditResult{Created: true, BytesWritten: len(newText)}, nil
	case err != nil:
		return nil, fmt.Errorf("edit: %w", err)
	}

	content := string(existing)
	if old == "" {
		updated := content + newText
		if err := os.WriteFile(full, []byte(updated), 0o644); err != nil {
			return nil, fmt.Errorf("edit: %w", err)
		}
		return &EditResult{BytesWritten: len(newText)}, nil
	}

	n := strings.Count(content, old)
	switch {
	case n == 0:
		return nil, fmt.Errorf("edit: old text not found in %s", path)
	case n > 1 && !replaceAll:
		return nil, fmt.Errorf("edit: old text appears %d times in %s; make it more specific or set replaceAll", n, path)
	}

	limit := 1
	if replaceAll {
		limit = -1
	}
	updated := strings.Replace(content, old, newText, limit)
	if err := os.WriteFile(full, []byte(updated), 0o644); err != nil {
		return nil, fmt.Errorf("edit: %w", err)
	}
	return &EditResult{BytesWritten: len(updated)}, nil
}
