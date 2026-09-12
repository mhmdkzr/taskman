package store

import "errors"

var (
	// ErrTaskNotFound is returned when no task exists for an id.
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskAlreadyExists is returned by Create when a task already exists.
	ErrTaskAlreadyExists = errors.New("task already exists")
	// errCorruptLog is returned when a stored event can't be decoded, e.g. an
	// unknown event kind.
	errCorruptLog = errors.New("task log is corrupt")
)
