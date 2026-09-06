// Package edit implements exact file content replacement.
package edit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	errPathRequired      = errors.New("path is required")
	errOldStringRequired = errors.New("old_string is required")
	errNoChange          = errors.New("old_string and new_string are identical")
	errNotFound          = errors.New("old_string not found in file")
	errNotUnique         = errors.New("old_string matches more than once; provide more context or set replace_all")
)

type Output struct {
	Path         string `json:"path"`
	Replacements int    `json:"replacements"`
}

func execute(_ context.Context, in Input) (Output, error) {
	info, err := os.Stat(in.Path)
	if err != nil {
		return Output{}, fmt.Errorf("edit: %w", err)
	}

	content, err := os.ReadFile(in.Path)
	if err != nil {
		return Output{}, fmt.Errorf("edit: %w", err)
	}

	count := strings.Count(string(content), in.OldString)
	if count == 0 {
		return Output{}, fmt.Errorf("edit: %w", errNotFound)
	}
	if count > 1 && !in.ReplaceAll {
		return Output{}, fmt.Errorf("edit: %w", errNotUnique)
	}

	replacements := 1
	replaceCount := 1
	if in.ReplaceAll {
		replacements = count
		replaceCount = -1
	}
	updated := strings.Replace(string(content), in.OldString, in.NewString, replaceCount)

	//nolint:gosec // The file-editing tool is explicitly designed to write the requested path.
	if err := os.WriteFile(in.Path, []byte(updated), info.Mode()); err != nil {
		return Output{}, fmt.Errorf("edit: %w", err)
	}

	return Output{Path: in.Path, Replacements: replacements}, nil
}
