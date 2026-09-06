// Package link manages links between persistent notes.
package link

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"log/slog"
	"strings"
	"time"
)

func execute(ctx context.Context, d tools.Deps, in Input) (Output, error) {
	if d.Store == nil {
		return Output{}, errors.New("notes_link: database is required")
	}
	if d.SessionID == (sessions.SessionID{}) {
		return Output{}, errors.New("notes_link: session is required")
	}
	from, to := strings.TrimSpace(in.From), strings.TrimSpace(in.To)
	tx, err := d.Store.RW().BeginTx(ctx, nil)
	if err != nil {
		return Output{}, fmt.Errorf("notes_link: begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("rollback notes link transaction", "error", err)
		}
	}()
	var a, b string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM notes WHERE name=? AND session_id=?`, from, d.SessionID.String()).Scan(&a); errors.Is(err, sql.ErrNoRows) {
		return Output{}, fmt.Errorf("notes_link: note %q not found", from)
	} else if err != nil {
		return Output{}, fmt.Errorf("notes_link: find %q: %w", from, err)
	}
	if err = tx.QueryRowContext(ctx, `SELECT id FROM notes WHERE name=? AND session_id=?`, to, d.SessionID.String()).Scan(&b); errors.Is(err, sql.ErrNoRows) {
		return Output{}, fmt.Errorf("notes_link: note %q not found", to)
	} else if err != nil {
		return Output{}, fmt.Errorf("notes_link: find %q: %w", to, err)
	}
	if b < a {
		a, b = b, a
	}
	rel := strings.TrimSpace(in.Relationship)
	if _, err = tx.ExecContext(ctx, `INSERT INTO notes_links(note_a,note_b,relationship,created_at) VALUES(?,?,?,?) ON CONFLICT(note_a,note_b) DO UPDATE SET relationship=excluded.relationship`, a, b, rel, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return Output{}, fmt.Errorf("notes_link: save link: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return Output{}, fmt.Errorf("notes_link: commit: %w", err)
	}
	return Output{From: from, To: to, Relationship: rel}, nil
}
