// Package delete implements persistent note deletion.
package delete

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
		return Output{}, errors.New("notes_delete: database is required")
	}
	if d.SessionID == (sessions.SessionID{}) {
		return Output{}, errors.New("notes_delete: session is required")
	}
	name := strings.TrimSpace(in.Name)
	tx, err := d.Store.RW().BeginTx(ctx, nil)
	if err != nil {
		return Output{}, fmt.Errorf("notes_delete: begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("rollback notes delete transaction", "error", err)
		}
	}()
	var id string
	err = tx.QueryRowContext(ctx, `SELECT n.id
		FROM notes n JOIN agent_sessions s ON s.session_id=n.session_id
		WHERE n.name=? AND n.session_id=? AND s.deleted_at IS NULL`, name, d.SessionID.String()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Output{}, errors.New("notes_delete: note not found")
	}
	if err != nil {
		return Output{}, fmt.Errorf("notes_delete: find note: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM notes WHERE id=?`, id); err != nil {
		return Output{}, fmt.Errorf("notes_delete: delete note: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return Output{}, fmt.Errorf("notes_delete: commit: %w", err)
	}
	return Output{Deleted: true, Name: name}, nil
}
