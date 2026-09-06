// Package write implements persistent note creation and replacement.
package write

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"uuid"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
)

func execute(ctx context.Context, d tools.Deps, in Input) (Output, error) {
	if d.Store == nil {
		return Output{}, errors.New("notes_write: database is required")
	}
	if d.SessionID == (sessions.SessionID{}) {
		return Output{}, errors.New("notes_write: session is required")
	}

	name := strings.TrimSpace(in.Name)
	body := strings.TrimSpace(in.Body)
	tx, err := d.Store.RW().BeginTx(ctx, nil)
	if err != nil {
		return Output{}, fmt.Errorf("notes_write: begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("rollback notes transaction", "error", err)
		}
	}()
	var sessionExists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM agent_sessions WHERE session_id = ? AND deleted_at IS NULL`, d.SessionID.String()).Scan(&sessionExists); errors.Is(err, sql.ErrNoRows) {
		return Output{}, errors.New("notes_write: session not found")
	} else if err != nil {
		return Output{}, fmt.Errorf("notes_write: check session: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := uuid.NewV7().String()
	var savedID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO notes (id, session_id, name, body, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET body = excluded.body
		RETURNING id`, id, d.SessionID.String(), name, body, now).Scan(&savedID)
	if err != nil {
		return Output{}, fmt.Errorf("notes_write: save note: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM notes_links WHERE note_a = ? OR note_b = ?`, savedID, savedID); err != nil {
		return Output{}, fmt.Errorf("notes_write: clear links: %w", err)
	}
	for _, requested := range in.Links {
		linkName := strings.TrimSpace(requested.Name)
		var otherID string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM notes WHERE name = ?`, linkName).Scan(&otherID); errors.Is(err, sql.ErrNoRows) {
			return Output{}, fmt.Errorf("notes_write: linked note %q not found", linkName)
		} else if err != nil {
			return Output{}, fmt.Errorf("notes_write: find linked note %q: %w", linkName, err)
		}

		a, b := savedID, otherID
		if b < a {
			a, b = b, a
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO notes_links (note_a, note_b, relationship, created_at)
			VALUES (?, ?, ?, ?)`, a, b, strings.TrimSpace(requested.Relationship), now); err != nil {
			return Output{}, fmt.Errorf("notes_write: save link to %q: %w", linkName, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Output{}, fmt.Errorf("notes_write: commit: %w", err)
	}

	links := make([]Link, len(in.Links))
	for i, requested := range in.Links {
		links[i] = Link{Name: strings.TrimSpace(requested.Name), Relationship: strings.TrimSpace(requested.Relationship)}
	}
	return Output{ID: savedID, Name: name, Body: body, Links: links}, nil
}
