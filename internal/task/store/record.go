package store

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mhmdkzr/taskman/internal/task"
)

func encodeEvent(event task.TaskEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("encode %s event: %w", event.Kind(), err)
	}
	return data, nil
}

// decodeEvent decodes data into the concrete task.TaskEvent matching kind,
// via task.DecodeEvent - the single place that maps EventKind to its Go
// type. An unrecognized kind becomes errCorruptLog, this package's own
// signal for an unreadable stored event; any other decode failure (a
// malformed payload for an otherwise-known kind) bubbles up as-is.
func decodeEvent(kind task.EventKind, data []byte) (task.TaskEvent, error) {
	event, err := task.DecodeEvent(kind, data)
	if err != nil {
		if errors.Is(err, task.ErrUnknownEventKind) {
			return nil, fmt.Errorf("%w: %w", errCorruptLog, err)
		}
		return nil, fmt.Errorf("decode event: %w", err)
	}
	return event, nil
}
