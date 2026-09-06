// Package store owns the read-write and read-only SQLite connections used by
// the application.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // Register the SQLite database driver.
)

const dirPerm = 0o700

// Store owns the two database handles used by the application. RW is intended
// for writes and reads that must immediately follow a write; RO is intended
// for unrestricted reads. Store is safe for concurrent use.
type Store struct {
	rw *sql.DB
	ro *sql.DB
}

// Open opens both database handles for path. The database must be file-backed;
// a second connection to :memory: would refer to a different database.
func Open(ctx context.Context, path string) (*Store, error) {
	rw, err := OpenDB(path)
	if err != nil {
		return nil, err
	}
	// sql.Open is lazy. Ping creates the file before the read-only handle opens.
	if err := rw.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("open: %w", errors.Join(err, rw.Close()))
	}
	ro, err := OpenReadOnly(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("open read-only: %w", errors.Join(err, rw.Close()))
	}
	return &Store{rw: rw, ro: ro}, nil
}

// RW returns the read-write database handle.
func (s *Store) RW() *sql.DB { return s.rw }

// RO returns the read-only database handle. SQLite rejects write statements
// executed through this handle.
func (s *Store) RO() *sql.DB { return s.ro }

// Close closes both database handles.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	if err := errors.Join(s.rw.Close(), s.ro.Close()); err != nil {
		return fmt.Errorf("close store: %w", err)
	}
	return nil
}

// dsn configures write transactions, lock waiting, and foreign-key checks on
// every connection in the write pool.
func dsn(path string) string {
	options := "_txlock=immediate&_busy_timeout=5000&_foreign_keys=on"
	if strings.ContainsRune(path, '?') {
		return path + "&" + options
	}
	return path + "?" + options
}

// OpenReadOnly opens a SQLite connection in mode=ro. The caller owns the
// returned handle and must close it.
func OpenReadOnly(ctx context.Context, path string) (*sql.DB, error) {
	if path == ":memory:" {
		return nil, fmt.Errorf("open read-only: :memory: is not supported; use a file path")
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_txlock=deferred&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open read-only database: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("open read-only: %w", errors.Join(err, db.Close()))
	}
	return db, nil
}

// OpenDB opens a read-write SQLite connection. It is kept separate from Open
// for callers such as migrations and in-memory database tests.
func OpenDB(path string) (*sql.DB, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	}
	return db, nil
}
