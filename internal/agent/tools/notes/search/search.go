// Package search searches persistent notes using SQLite FTS5.
package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
)

func execute(ctx context.Context, d tools.Deps, in Input) (Output, error) {
	if d.Store == nil {
		return Output{}, errors.New("notes_search: database is required")
	}
	if d.SessionID == (sessions.SessionID{}) {
		return Output{}, errors.New("notes_search: session is required")
	}
	limit := 20
	if in.Limit != nil {
		limit = *in.Limit
	}
	rows, err := d.Store.RO().QueryContext(ctx, `SELECT n.id,n.name,n.body,n.created_at
		FROM notes_fts f JOIN notes n ON n.rowid=f.rowid
		JOIN agent_sessions s ON s.session_id=n.session_id
		WHERE notes_fts MATCH ? AND n.session_id=? AND s.deleted_at IS NULL
		ORDER BY rank LIMIT ?`, strings.TrimSpace(in.Query), d.SessionID.String(), limit)
	if err != nil {
		return Output{}, fmt.Errorf("notes_search: query notes: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.Error("close note search rows", "error", closeErr)
		}
	}()
	out := Output{Notes: []Note{}, Limit: limit}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Name, &n.Body, &n.CreatedAt); err != nil {
			return Output{}, fmt.Errorf("notes_search: scan note: %w", err)
		}
		out.Notes = append(out.Notes, n)
	}
	if err := rows.Err(); err != nil {
		return Output{}, fmt.Errorf("notes_search: read notes: %w", err)
	}
	return out, nil
}
