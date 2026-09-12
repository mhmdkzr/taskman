// Package store is the event-sourced, SQLite-backed persistence for tasks.
// A task's current state is never stored directly - it is always derived by
// replaying its events, in order, through task.Apply.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	sqlite "modernc.org/sqlite"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
)

const schema = `
CREATE TABLE IF NOT EXISTS tasks (
	id         TEXT PRIMARY KEY,
	definition TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS events (
	task_id TEXT    NOT NULL REFERENCES tasks(id),
	seq     INTEGER NOT NULL,
	kind    TEXT    NOT NULL,
	data    TEXT    NOT NULL,
	PRIMARY KEY (task_id, seq)
);
`

// sqliteConstraintResultCode is SQLITE_CONSTRAINT's base result code; every
// extended constraint code (primary key, unique, foreign key, ...) shares it
// in its low byte, so checking against it covers all of them without
// depending on modernc.org/sqlite's internal extended-code constants.
const sqliteConstraintResultCode = 19

// Store is a task/store handle: a single SQLite database holding every
// task's identity and event log.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path and ensures
// its schema exists.
func Open(path string) (*Store, error) {
	dsn := path + "?" + url.Values{
		"_journal_mode": {"WAL"},
		"_busy_timeout": {"5000"},
		"_pragma":       {"foreign_keys(1)"},
	}.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate store: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the store's database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// Create starts a new task and returns it. It fails with
// ErrTaskAlreadyExists if id is already present.
func (s *Store) Create(id uuid.UUID, definition task.TaskDefinition, at time.Time) (task.Task, error) {
	current, err := task.NewTask(id, definition, at)
	if err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}

	def, err := json.Marshal(definition)
	if err != nil {
		return task.Task{}, fmt.Errorf("create task %s: %w", id, err)
	}

	_, err = s.db.ExecContext(context.Background(),
		`INSERT INTO tasks (id, definition, created_at) VALUES (?, ?, ?)`,
		id.String(), string(def), at.Format(time.RFC3339Nano),
	)
	if err != nil {
		if isConstraintViolation(err) {
			return task.Task{}, fmt.Errorf("create task %s: %w", id, ErrTaskAlreadyExists)
		}
		return task.Task{}, fmt.Errorf("create task %s: %w", id, err)
	}
	return current, nil
}

// Read replays id's events and returns the resulting task.
func (s *Store) Read(id uuid.UUID) (task.Task, error) {
	return s.read(context.Background(), s.db, id)
}

// Append validates event against id's current (replayed) task and, if
// accepted, durably appends it as the next event, returning the resulting
// task. It never persists a rejected event. The whole read-validate-append
// cycle runs in one BEGIN IMMEDIATE transaction, so concurrent Appends
// serialize instead of racing on a stale read.
func (s *Store) Append(id uuid.UUID, event task.TaskEvent) (task.Task, error) {
	ctx := context.Background()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return task.Task{}, fmt.Errorf("append task %s: %w", id, err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return task.Task{}, fmt.Errorf("append task %s: %w", id, err)
	}

	next, err := s.appendLocked(ctx, conn, id, event)
	if err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return task.Task{}, err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return task.Task{}, fmt.Errorf("append task %s: %w", id, err)
	}
	return next, nil
}

func (s *Store) appendLocked(ctx context.Context, conn *sql.Conn, id uuid.UUID, event task.TaskEvent) (task.Task, error) {
	current, err := s.read(ctx, conn, id)
	if err != nil {
		return task.Task{}, err
	}
	next, err := task.Apply(current, event)
	if err != nil {
		return task.Task{}, err
	}

	data, err := encodeEvent(event)
	if err != nil {
		return task.Task{}, fmt.Errorf("append task %s: %w", id, err)
	}
	_, err = conn.ExecContext(ctx,
		`INSERT INTO events (task_id, seq, kind, data)
		 VALUES (?, (SELECT COALESCE(MAX(seq), 0) + 1 FROM events WHERE task_id = ?), ?, ?)`,
		id.String(), id.String(), string(event.Kind()), string(data),
	)
	if err != nil {
		return task.Task{}, fmt.Errorf("append task %s: %w", id, err)
	}
	return next, nil
}

// List returns the ids of every task in the store.
func (s *Store) List() ([]uuid.UUID, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id FROM tasks ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var idStr string
		if err := rows.Scan(&idStr); err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return ids, nil
}

// querier is satisfied by both *sql.DB and *sql.Conn, so read works whether
// or not it's inside Append's transaction.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s *Store) read(ctx context.Context, q querier, id uuid.UUID) (task.Task, error) {
	var defJSON, createdAt string
	err := q.QueryRowContext(ctx, `SELECT definition, created_at FROM tasks WHERE id = ?`, id.String()).
		Scan(&defJSON, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return task.Task{}, ErrTaskNotFound
		}
		return task.Task{}, fmt.Errorf("read task %s: %w", id, err)
	}
	var definition task.TaskDefinition
	if err := json.Unmarshal([]byte(defJSON), &definition); err != nil {
		return task.Task{}, fmt.Errorf("decode task %s definition: %w", id, err)
	}
	at, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return task.Task{}, fmt.Errorf("decode task %s created_at: %w", id, err)
	}
	current, err := task.NewTask(id, definition, at)
	if err != nil {
		return task.Task{}, fmt.Errorf("task %s: %w", id, err)
	}

	rows, err := q.QueryContext(ctx, `SELECT kind, data FROM events WHERE task_id = ? ORDER BY seq`, id.String())
	if err != nil {
		return task.Task{}, fmt.Errorf("read task %s events: %w", id, err)
	}
	defer rows.Close()

	for rows.Next() {
		var kind, data string
		if err := rows.Scan(&kind, &data); err != nil {
			return task.Task{}, fmt.Errorf("read task %s events: %w", id, err)
		}
		event, err := decodeEvent(task.EventKind(kind), []byte(data))
		if err != nil {
			return task.Task{}, fmt.Errorf("decode task %s event: %w", id, err)
		}
		current, err = task.Apply(current, event)
		if err != nil {
			return task.Task{}, fmt.Errorf("replay task %s: %w", id, err)
		}
	}
	if err := rows.Err(); err != nil {
		return task.Task{}, fmt.Errorf("read task %s events: %w", id, err)
	}
	return current, nil
}

// isConstraintViolation reports whether err is a SQLite constraint failure
// (e.g. a duplicate primary key on tasks.id).
func isConstraintViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code()&0xff == sqliteConstraintResultCode
	}
	return false
}
