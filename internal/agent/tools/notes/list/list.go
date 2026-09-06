// Package list lists persistent notes.
package list

import (
	"context"
	"errors"
	"fmt"
	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
)

func execute(ctx context.Context, d tools.Deps, in Input) (Output, error) {
	if d.Store == nil {
		return Output{}, errors.New("notes_list: database is required")
	}
	if d.SessionID == (sessions.SessionID{}) {
		return Output{}, errors.New("notes_list: session is required")
	}
	limit := 20
	if in.Limit != nil {
		limit = *in.Limit
	}
	rows, err := d.Store.RO().QueryContext(ctx, `SELECT n.id,n.name,n.body,n.created_at FROM notes n JOIN agent_sessions s ON s.session_id=n.session_id WHERE n.session_id=? AND s.deleted_at IS NULL ORDER BY n.created_at DESC,n.id DESC LIMIT ?`, d.SessionID.String(), limit)
	if err != nil {
		return Output{}, fmt.Errorf("notes_list: query notes: %w", err)
	}
	defer rows.Close()
	out := Output{Notes: []Note{}, Limit: limit}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Name, &n.Body, &n.CreatedAt); err != nil {
			return Output{}, fmt.Errorf("notes_list: scan note: %w", err)
		}
		out.Notes = append(out.Notes, n)
	}
	if err := rows.Err(); err != nil {
		return Output{}, fmt.Errorf("notes_list: read notes: %w", err)
	}
	return out, nil
}
