// Package history provides read-only search over past conversation turns
// across agent sessions.
package history

import (
	"context"
	"fmt"
	"strings"
	"uuid"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/store"
)

const (
	// defaultLimit is the number of turns returned when limit is unset.
	defaultLimit = 10
	// maxLimit bounds limit so a single read cannot flood the context window.
	maxLimit = 50
	// defaultCellLen is the number of characters rendered per prompt/reply
	// when max_cell_len is unset.
	defaultCellLen = 200
	// maxCellLen bounds max_cell_len.
	maxCellLen = 10000
)

func execute(ctx context.Context, st *store.Store, in Input) (Output, error) {
	if st == nil {
		return Output{}, fmt.Errorf("database is required")
	}

	limit := defaultLimit
	if in.Limit != nil {
		limit = *in.Limit
	}
	cellLen := defaultCellLen
	if in.MaxCellLen != nil {
		cellLen = *in.MaxCellLen
	}

	var sessionID *sessions.SessionID
	if trimmed := strings.TrimSpace(in.SessionID); trimmed != "" {
		id, err := uuid.Parse(trimmed)
		if err != nil {
			return Output{}, fmt.Errorf("history: invalid session_id: %w", err)
		}
		sid := sessions.SessionID(id)
		sessionID = &sid
	}

	turns, err := sessions.SearchHistory(ctx, st, sessionID, strings.TrimSpace(in.Query), limit)
	if err != nil {
		return Output{}, fmt.Errorf("history: %w", err)
	}
	if len(turns) == 0 {
		return Output{History: "no conversation history found"}, nil
	}
	return Output{History: formatHistory(turns, cellLen)}, nil
}

// formatHistory renders turns grouped by session. Long prompts and replies
// are truncated to cellLen so a read cannot flood the context window.
func formatHistory(turns []sessions.HistoryTurn, cellLen int) string {
	render := func(text string) (string, bool) {
		if len(text) <= cellLen {
			return text, false
		}
		return text[:cellLen] + "...", true
	}

	var b strings.Builder
	truncated := 0
	lastSession := ""
	for _, t := range turns {
		sessionID := t.SessionID.String()
		if sessionID != lastSession {
			fmt.Fprintf(&b, "session %s (created %s)\n", sessionID, t.SessionCreatedAt)
			lastSession = sessionID
		}
		prompt, promptTruncated := render(t.Prompt)
		reply, replyTruncated := render(t.Reply)
		if promptTruncated {
			truncated++
		}
		if replyTruncated {
			truncated++
		}
		fmt.Fprintf(&b, "  [%s] user: %s\n", t.CreatedAt, prompt)
		fmt.Fprintf(&b, "  [%s] assistant: %s\n", t.CreatedAt, reply)
	}

	out := strings.TrimSuffix(b.String(), "\n")
	if truncated > 0 {
		out += fmt.Sprintf("\n(... %d messages truncated to %d characters; use the query tool to read them in full)",
			truncated, cellLen)
	}
	return out
}
