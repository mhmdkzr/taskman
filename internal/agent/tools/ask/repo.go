package ask

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"
)

// ErrAskNotFound is returned when no pending ask matches the requested ID.
var ErrAskNotFound = errors.New("ask not found")

// PendingAsk is a still-unanswered question, as a transcript view renders it.
type PendingAsk struct {
	AskID       uuid.UUID
	Question    string
	Options     []Option
	MultiSelect bool
}

// createAsk persists a new pending question against turnID, which the ask
// tool call then polls (see execute) until AnswerAsk resolves it.
func createAsk(
	ctx context.Context,
	db *sql.DB,
	askID uuid.UUID,
	sessionID, turnID string,
	in Input,
) error {
	optionsJSON, err := json.Marshal(in.Options)
	if err != nil {
		return fmt.Errorf("marshal ask options: %w", err)
	}
	multiSelect := 0
	if in.MultiSelect {
		multiSelect = 1
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO session_asks (
			ask_id, session_id, turn_id, question, options, multi_select, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		askID.String(), sessionID, turnID, in.Question, string(optionsJSON), multiSelect,
		time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert ask: %w", err)
	}
	return nil
}

// pollAnswer reports whether askID has been answered yet, and its answer if so.
func pollAnswer(ctx context.Context, db *sql.DB, askID uuid.UUID) (bool, []int, string, error) {
	var (
		status      string
		selectedRaw sql.NullString
		customRaw   sql.NullString
	)
	err := db.QueryRowContext(ctx, `
		SELECT status, selected, custom FROM session_asks WHERE ask_id = ?`, askID.String()).
		Scan(&status, &selectedRaw, &customRaw)
	if err != nil {
		return false, nil, "", fmt.Errorf("query ask: %w", err)
	}
	if status != "answered" {
		return false, nil, "", nil
	}
	var selected []int
	if selectedRaw.Valid && selectedRaw.String != "" {
		if err := json.Unmarshal([]byte(selectedRaw.String), &selected); err != nil {
			return false, nil, "", fmt.Errorf("decode ask selected: %w", err)
		}
	}
	return true, selected, customRaw.String, nil
}

// runningTurnID returns the turn currently in progress for sessionID. A
// session runs one turn at a time (sessions.Run is synchronous per session),
// so at most one row can match.
func runningTurnID(ctx context.Context, db *sql.DB, sessionID string) (string, error) {
	var turnID string
	err := db.QueryRowContext(ctx, `
		SELECT turn_id FROM session_turns WHERE session_id = ? AND status = 'running'
		ORDER BY created_at DESC LIMIT 1`, sessionID).Scan(&turnID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("no running turn for session %s", sessionID)
	}
	if err != nil {
		return "", fmt.Errorf("query running turn: %w", err)
	}
	return turnID, nil
}

// AnswerAsk records a human's answer to a pending ask, which the blocked ask
// tool call picks up on its next poll (see execute). Answering an ask that is
// already answered or doesn't exist is an error: each ask is answered once.
func AnswerAsk(ctx context.Context, db *sql.DB, askID uuid.UUID, selected []int, custom string) error {
	selectedJSON, err := json.Marshal(selected)
	if err != nil {
		return fmt.Errorf("marshal selected: %w", err)
	}
	res, err := db.ExecContext(ctx, `
		UPDATE session_asks
		SET status = 'answered', selected = ?, custom = ?, answered_at = ?
		WHERE ask_id = ? AND status = 'pending'`,
		string(selectedJSON), custom, time.Now().UTC().Format(time.RFC3339Nano), askID.String())
	if err != nil {
		return fmt.Errorf("update ask: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrAskNotFound
	}
	return nil
}

// PendingAskForTurn returns turnID's still-unanswered ask, if any - what a
// transcript view renders as an interactive question card while a turn with
// a pending ask is running. It returns (nil, nil) when there is none.
func PendingAskForTurn(ctx context.Context, db *sql.DB, turnID string) (*PendingAsk, error) {
	var (
		askIDStr    string
		question    string
		optionsRaw  sql.NullString
		multiSelect int
	)
	err := db.QueryRowContext(ctx, `
		SELECT ask_id, question, options, multi_select
		FROM session_asks
		WHERE turn_id = ? AND status = 'pending'
		ORDER BY created_at DESC LIMIT 1`, turnID).
		Scan(&askIDStr, &question, &optionsRaw, &multiSelect)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil //nolint:nilnil // no pending ask is a normal outcome, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("query pending ask: %w", err)
	}

	askID, err := uuid.Parse(askIDStr)
	if err != nil {
		return nil, fmt.Errorf("parse ask id: %w", err)
	}
	var options []Option
	if optionsRaw.Valid && optionsRaw.String != "" {
		if err := json.Unmarshal([]byte(optionsRaw.String), &options); err != nil {
			return nil, fmt.Errorf("decode ask options: %w", err)
		}
	}
	return &PendingAsk{
		AskID:       askID,
		Question:    question,
		Options:     options,
		MultiSelect: multiSelect == 1,
	}, nil
}
