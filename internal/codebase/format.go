package codebase

import (
	"errors"
	"fmt"
)

// Format applies gofmt and goimports in place. It runs both regardless of
// whether one fails, and joins their errors (nil if both succeed) — a
// pipeline step, not something that needs agent judgment.
func (r Repository) Format() error {
	fmtErr := r.GoFmt()
	impErr := r.GoImports()
	if err := errors.Join(fmtErr, impErr); err != nil {
		return fmt.Errorf("format: %w", err)
	}
	return nil
}
