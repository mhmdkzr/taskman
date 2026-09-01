package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Scheduled mirrors a row of scheduler: a one-time message to publish on a NATS
// subject once its publish_at deadline passes. CreatedAt mirrors the timestamp
// embedded in the UUIDv7 id, so the id and the scheduling time are always in
// sync; PublishAt is the caller-supplied target publish time; PublishedAt is
// set once the payload is actually sent. RecurrenceID links an occurrence to
// its parent recurrence; it is empty for one-shot messages.
type Scheduled struct {
	ID           string
	Subject      string
	Payload      string
	CreatedAt    string
	PublishAt    int64
	Status       string
	PublishedAt  sql.NullString
	RecurrenceID sql.NullString
}

// Recurrence mirrors a row of recurrences: a fixed-interval repeating schedule.
// Each occurrence is a separate scheduler row linked by recurrence_id, so
// delivery reuses the one-shot exactly-once machinery. Status is active until
// cancelled.
type Recurrence struct {
	ID         string
	Subject    string
	Payload    string
	IntervalMs int64
	CreatedAt  string
	Status     string
}

// executor is the common interface of *sql.DB and *sql.Tx, so inserts can run
// standalone or inside a transaction.
type executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// CreateScheduled inserts a one-time scheduled message and returns it. The id
// is a fresh UUIDv7 whose embedded timestamp is stored as created_at; the
// payload is published once publishAt passes.
func CreateScheduled(ctx context.Context, db *sql.DB, subject, payload string, publishAt time.Time) (*Scheduled, error) {
	return createScheduled(ctx, db, subject, payload, publishAt, "")
}

// CreateScheduledOccurrence inserts the next occurrence of a recurrence,
// linking the row to its parent so firing or skipping it advances the schedule.
func CreateScheduledOccurrence(ctx context.Context, db *sql.DB, subject, payload string, publishAt time.Time, recurrenceID string) (*Scheduled, error) {
	return createScheduled(ctx, db, subject, payload, publishAt, recurrenceID)
}

func createScheduled(ctx context.Context, db executor, subject, payload string, publishAt time.Time, recurrenceID string) (*Scheduled, error) {
	id, createdAt := newIDWithTime()
	s := &Scheduled{
		ID:        id,
		Subject:   subject,
		Payload:   payload,
		CreatedAt: createdAt,
		PublishAt: publishAt.UnixMilli(),
		Status:    "pending",
	}
	if recurrenceID != "" {
		s.RecurrenceID = sql.NullString{String: recurrenceID, Valid: true}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO scheduler (id, subject, payload, created_at, publish_at, status, recurrence_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Subject, s.Payload, s.CreatedAt, s.PublishAt, s.Status, nullStr(s.RecurrenceID.String)); err != nil {
		return nil, fmt.Errorf("insert scheduled: %w", err)
	}
	return s, nil
}

// CreateRecurrence inserts a repeating schedule and returns it. The first
// occurrence is not created here; the caller schedules it separately so the
// first run is under the caller's control.
func CreateRecurrence(ctx context.Context, db *sql.DB, subject, payload string, interval time.Duration) (*Recurrence, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("create recurrence: interval must be positive")
	}
	id, createdAt := newIDWithTime()
	r := &Recurrence{
		ID:         id,
		Subject:    subject,
		Payload:    payload,
		IntervalMs: interval.Milliseconds(),
		CreatedAt:  createdAt,
		Status:     "active",
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO recurrences (id, subject, payload, interval_ms, created_at, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Subject, r.Payload, r.IntervalMs, r.CreatedAt, r.Status); err != nil {
		return nil, fmt.Errorf("create recurrence: %w", err)
	}
	return r, nil
}

// ListPending returns every message that has not yet fired or expired, soonest
// deadline first. Used to arm timers on startup.
func ListPending(ctx context.Context, db *sql.DB) ([]Scheduled, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, subject, payload, created_at, publish_at, status, published_at, recurrence_id
		FROM scheduler WHERE status = 'pending' ORDER BY publish_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list pending: %w", err)
	}
	defer rows.Close()
	return scanScheduledRows(rows)
}

// ListDue returns pending messages whose publish deadline has passed, soonest
// deadline first. A message is due once publish_at <= nowMs; whether it fires
// or expires is the scheduler's call (via the grace period).
func ListDue(ctx context.Context, db *sql.DB, nowMs int64) ([]Scheduled, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, subject, payload, created_at, publish_at, status, published_at, recurrence_id
		FROM scheduler WHERE status = 'pending' AND publish_at <= ?
		ORDER BY publish_at ASC`, nowMs)
	if err != nil {
		return nil, fmt.Errorf("list due: %w", err)
	}
	defer rows.Close()
	return scanScheduledRows(rows)
}

// ListUnpublished returns fired messages that were never marked published. A
// crash between claiming a message and recording its publish leaves such a row;
// the scheduler replays it for at-least-once delivery.
func ListUnpublished(ctx context.Context, db *sql.DB) ([]Scheduled, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, subject, payload, created_at, publish_at, status, published_at, recurrence_id
		FROM scheduler WHERE status = 'fired' AND published_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("list unpublished: %w", err)
	}
	defer rows.Close()
	return scanScheduledRows(rows)
}

// ListScheduled returns every scheduler row, pending first (soonest deadline
// first), then the terminal rows (fired, expired, cancelled) by deadline. It is
// the full view an agent needs to decide which scheduled messages to cancel.
func ListScheduled(ctx context.Context, db *sql.DB) ([]Scheduled, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, subject, payload, created_at, publish_at, status, published_at, recurrence_id
		FROM scheduler ORDER BY CASE status WHEN 'pending' THEN 0 ELSE 1 END, publish_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list scheduled: %w", err)
	}
	defer rows.Close()
	return scanScheduledRows(rows)
}

// ListRecurrences returns every recurrence, active first, then cancelled, each
// by creation order. It is the view an agent needs to see what is scheduled to
// repeat and what has been stopped.
func ListRecurrences(ctx context.Context, db *sql.DB) ([]Recurrence, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, subject, payload, interval_ms, created_at, status
		FROM recurrences ORDER BY CASE status WHEN 'active' THEN 0 ELSE 1 END, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list recurrences: %w", err)
	}
	defer rows.Close()
	var out []Recurrence
	for rows.Next() {
		var r Recurrence
		if err := rows.Scan(&r.ID, &r.Subject, &r.Payload, &r.IntervalMs, &r.CreatedAt, &r.Status); err != nil {
			return nil, fmt.Errorf("scan recurrence: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// CancelScheduled flips a pending message to cancelled. It reports whether this
// call won; a lost cancel means the message was already fired, expired or
// cancelled, so there is nothing left to withdraw.
func CancelScheduled(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `UPDATE scheduler SET status = 'cancelled'
		WHERE id = ? AND status = 'pending'`, id)
	if err != nil {
		return false, fmt.Errorf("cancel scheduled %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// CancelRecurrence flips a recurrence from active to cancelled and, in the same
// transaction, cancels its pending next occurrence so no future run fires. It
// reports whether this call won; a lost cancel means the recurrence was already
// cancelled.
func CancelRecurrence(ctx context.Context, db *sql.DB, id string) (bool, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("cancel recurrence %s: %w", id, err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `UPDATE recurrences SET status = 'cancelled'
		WHERE id = ? AND status = 'active'`, id)
	if err != nil {
		return false, fmt.Errorf("cancel recurrence %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n != 1 {
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE scheduler SET status = 'cancelled'
		WHERE recurrence_id = ? AND status = 'pending'`, id); err != nil {
		return false, fmt.Errorf("cancel recurrence %s: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// ClaimAndAdvance claims a pending occurrence, flipping it to a terminal status
// ('fired' or 'expired'). When the occurrence belongs to an active recurrence,
// the next occurrence (at publish_at + interval_ms) is inserted in the same
// transaction and returned, so exactly one scheduler advances the schedule: a
// lost claim (the row was already handled) neither publishes nor advances.
func ClaimAndAdvance(ctx context.Context, db *sql.DB, id, status string) (won bool, next *Scheduled, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, fmt.Errorf("claim and advance %s: %w", id, err)
	}
	defer tx.Rollback()

	var s Scheduled
	if err := tx.QueryRowContext(ctx, `SELECT id, subject, payload, created_at, publish_at, status, published_at, recurrence_id
		FROM scheduler WHERE id = ?`, id).
		Scan(&s.ID, &s.Subject, &s.Payload, &s.CreatedAt, &s.PublishAt, &s.Status, &s.PublishedAt, &s.RecurrenceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("claim and advance %s: %w", id, err)
	}

	res, err := tx.ExecContext(ctx, `UPDATE scheduler SET status = ?
		WHERE id = ? AND status = 'pending'`, status, id)
	if err != nil {
		return false, nil, fmt.Errorf("claim and advance %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, nil, err
	}
	if n != 1 {
		return false, nil, nil
	}

	if s.RecurrenceID.Valid {
		var recStatus string
		var intervalMs int64
		if err := tx.QueryRowContext(ctx, `SELECT status, interval_ms FROM recurrences WHERE id = ?`,
			s.RecurrenceID.String).Scan(&recStatus, &intervalMs); err == nil && recStatus == "active" {
			occurrence, err := createScheduled(ctx, tx, s.Subject, s.Payload,
				time.UnixMilli(s.PublishAt+intervalMs), s.RecurrenceID.String)
			if err != nil {
				return false, nil, fmt.Errorf("claim and advance %s: %w", id, err)
			}
			next = occurrence
		}
	}

	if err := tx.Commit(); err != nil {
		return false, nil, err
	}
	return true, next, nil
}

// ClaimForFire flips a pending message to fired. It reports whether this call
// won the claim; a lost claim means another scheduler (or the poll vs. a timer)
// already handled the message.
func ClaimForFire(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `UPDATE scheduler SET status = 'fired'
		WHERE id = ? AND status = 'pending'`, id)
	if err != nil {
		return false, fmt.Errorf("claim for fire %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// ClaimExpired flips a pending message to expired. It reports whether this call
// won the claim; a lost claim means the message was already handled.
func ClaimExpired(ctx context.Context, db *sql.DB, id string) (bool, error) {
	res, err := db.ExecContext(ctx, `UPDATE scheduler SET status = 'expired'
		WHERE id = ? AND status = 'pending'`, id)
	if err != nil {
		return false, fmt.Errorf("claim expired %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// MarkPublished records when a message's payload was actually published.
func MarkPublished(ctx context.Context, db *sql.DB, id string) error {
	if _, err := db.ExecContext(ctx, `UPDATE scheduler SET published_at = ?
		WHERE id = ?`, time.Now().UTC().String(), id); err != nil {
		return fmt.Errorf("mark published %s: %w", id, err)
	}
	return nil
}

func scanScheduledRows(rows *sql.Rows) ([]Scheduled, error) {
	var out []Scheduled
	for rows.Next() {
		var s Scheduled
		if err := rows.Scan(&s.ID, &s.Subject, &s.Payload, &s.CreatedAt, &s.PublishAt, &s.Status, &s.PublishedAt, &s.RecurrenceID); err != nil {
			return nil, fmt.Errorf("scan scheduled: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
