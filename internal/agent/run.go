package agent

import (
	"context"

	"github.com/mhmdkzr/taskman/internal/publisher"
)

// Run executes a single prompt non-streaming and returns the final result. It
// creates a fresh session with no prior history. sessionID pre-assigns the
// session id (e.g. so a run-started event can carry it); pass "" to generate a
// new id at session start. The id is available on the returned Result.
func Run(ctx context.Context, opts Options, pub publisher.Publisher, prompt, sessionID string) (*Result, error) {
	s, err := NewSession(opts, pub, sessionID)
	if err != nil {
		return nil, err
	}
	return s.Run(ctx, prompt)
}

// Stream starts a single prompt and returns an event stream. It creates a
// fresh session (with a new session id) and no prior history. Consume the
// stream's Events to completion, then check Err and call Result.
func Stream(ctx context.Context, opts Options, pub publisher.Publisher, prompt string) (*EventStream, error) {
	s, err := NewSession(opts, pub, "")
	if err != nil {
		return nil, err
	}
	return s.Stream(ctx, prompt)
}
