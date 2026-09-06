package ask

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/mhmdkzr/loop/internal/store"
)

// pollInterval is how often execute checks whether a pending ask has been
// answered yet. It only needs to be responsive enough for a human waiting on
// their own browser tab, not tuned for throughput.
const pollInterval = 500 * time.Millisecond

// execute persists in as a pending question against sessionID's currently
// running turn, then blocks - polling the database, per this package's
// stateless convention (see internal/agent/sessions) - until a human answers
// it via the web UI (AnswerAsk) or ctx is cancelled. A crash or process
// restart while blocked here leaves the turn "running", exactly like any
// other tool call in progress: sessions.ReconcileInterrupted recovers it the
// same way at next boot, synthesizing an "interrupted" result for this call
// like any other unattempted-or-unknown one.
func execute(ctx context.Context, st *store.Store, sessionID string, in Input) (Output, error) {
	turnID, err := runningTurnID(ctx, st.RO(), sessionID)
	if err != nil {
		return Output{}, fmt.Errorf("ask: %w", err)
	}

	askID := uuid.NewV7()
	if err := createAsk(ctx, st.RW(), askID, sessionID, turnID, in); err != nil {
		return Output{}, fmt.Errorf("ask: %w", err)
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return Output{}, fmt.Errorf("ask: %w", ctx.Err())
		case <-ticker.C:
			answered, selected, custom, err := pollAnswer(ctx, st.RO(), askID)
			if err != nil {
				return Output{}, fmt.Errorf("ask: %w", err)
			}
			if answered {
				return Output{Selected: selected, Custom: custom}, nil
			}
		}
	}
}
