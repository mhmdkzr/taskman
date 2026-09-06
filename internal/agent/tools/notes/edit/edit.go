// Package edit updates existing persistent notes.
package edit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
)

func execute(ctx context.Context, d tools.Deps, in Input) (Output, error) {
	if d.Store == nil {
		return Output{}, errors.New("notes_edit: database is required")
	}
	if d.SessionID == (sessions.SessionID{}) {
		return Output{}, errors.New("notes_edit: session is required")
	}
	name, body := strings.TrimSpace(in.Name), strings.TrimSpace(in.Body)
	var out Output
	err := d.Store.RW().QueryRowContext(ctx, `UPDATE notes SET body=?
		WHERE name=? AND session_id=? AND EXISTS (
			SELECT 1 FROM agent_sessions WHERE session_id=? AND deleted_at IS NULL
		) RETURNING id,name,body,created_at`, body, name, d.SessionID.String(), d.SessionID.String()).Scan(
		&out.ID, &out.Name, &out.Body, &out.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Output{}, errors.New("notes_edit: note not found")
	}
	if err != nil {
		return Output{}, fmt.Errorf("notes_edit: update note: %w", err)
	}
	return out, nil
}
