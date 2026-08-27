// Package temporalerr provides helpers for adding context to workflow errors without
// breaking Temporal's typed application error propagation.
package temporalerr

import (
	"errors"
	"fmt"

	"go.temporal.io/sdk/temporal"
)

// Wrap adds context to err, except typed application errors: %w-wrapping a
// *temporal.ApplicationError makes the workflow failure serialize with Type
// "wrapError", so clients could no longer branch on the caller's own type strings
// (e.g. ValidationError, InsufficientFunds).
func Wrap(context string, err error) error {
	if _, ok := errors.AsType[*temporal.ApplicationError](err); ok {
		return err
	}
	return fmt.Errorf("%s: %w", context, err)
}
