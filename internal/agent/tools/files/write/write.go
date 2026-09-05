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

type output struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

func execute(_ context.Context, in input) (output, error) {
	mode := os.FileMode(defaultMode)
	if info, err := os.Stat(in.Path); err == nil {
		mode = info.Mode()
	} else if !errors.Is(err, os.ErrNotExist) {
		return output{}, fmt.Errorf("write: %w", err)
	}

	if err := os.WriteFile(in.Path, []byte(in.Content), mode); err != nil {
		return output{}, fmt.Errorf("write: %w", err)
	}

	return output{Path: in.Path, Bytes: len(in.Content)}, nil
}
