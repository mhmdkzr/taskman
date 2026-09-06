// Package write implements file creation and replacement.
package write

import (
	"context"
	"errors"
	"fmt"
	"os"
)

const defaultMode = 0o644

var errPathRequired = errors.New("path is required")

type Output struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

func execute(_ context.Context, in Input) (Output, error) {
	mode := os.FileMode(defaultMode)
	if info, err := os.Stat(in.Path); err == nil {
		mode = info.Mode()
	} else if !errors.Is(err, os.ErrNotExist) {
		return Output{}, fmt.Errorf("write: %w", err)
	}

	if err := os.WriteFile(in.Path, []byte(in.Content), mode); err != nil {
		return Output{}, fmt.Errorf("write: %w", err)
	}

	return Output{Path: in.Path, Bytes: len(in.Content)}, nil
}
