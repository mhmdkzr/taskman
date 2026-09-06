// Package read reads persistent notes and their links.
package read

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
)

func execute(ctx context.Context, d tools.Deps, in Input) (Output, error) {
	if d.Store == nil {
		return Output{}, errors.New("notes_read: database is required")
	}
	if d.SessionID == (sessions.SessionID{}) {
		return Output{}, errors.New("notes_read: session is required")
	}
	var out Output
	name := strings.TrimSpace(in.Name)
	err := d.Store.RO().QueryRowContext(ctx, `SELECT n.id,n.name,n.body,n.created_at
		FROM notes n JOIN agent_sessions s ON s.session_id=n.session_id
		WHERE n.name=? AND n.session_id=? AND s.deleted_at IS NULL`, name, d.SessionID.String()).Scan(
		&out.ID, &out.Name, &out.Body, &out.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Output{}, errors.New("notes_read: note not found")
	}
	if err != nil {
		return Output{}, fmt.Errorf("notes_read: find note: %w", err)
	}
	rows, err := d.Store.RO().QueryContext(ctx, `SELECT n.name,l.relationship
		FROM notes_links l JOIN notes n ON n.id=CASE WHEN l.note_a=? THEN l.note_b ELSE l.note_a END
		WHERE l.note_a=? OR l.note_b=? ORDER BY n.name`, out.ID, out.ID, out.ID)
	if err != nil {
		return Output{}, fmt.Errorf("notes_read: query links: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.Error("close note links rows", "error", closeErr)
		}
	}()
	out.Links = []Link{}
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.Name, &l.Relationship); err != nil {
			return Output{}, fmt.Errorf("notes_read: scan link: %w", err)
		}
		out.Links = append(out.Links, l)
	}
	if err := rows.Err(); err != nil {
		return Output{}, fmt.Errorf("notes_read: read links: %w", err)
	}
	return out, nil
}
