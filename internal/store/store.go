package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const dirPerm = 0o700

// Store owns the two database handles an agent run uses: rw for writes (and
// reads that must not race a just-issued write), and ro for unrestricted
// read-only access. The read-only handle is opened with mode=ro, so statements
// executed through it are rejected by the storage engine before they can
// modify anything. Open one Store per process and close it with Close. Store
// is safe for concurrent use.
type Store struct {
	rw *sql.DB
	ro *sql.DB
}

// Open opens the read-write and read-only handles for the database at path and
// returns a Store owning both. The handles stay open until Close. In-memory
// databases are not supported: a read-only handle to ":memory:" would be a
// separate, empty database, so use a file path.
func Open(path string) (*Store, error) {
	rw, err := OpenDB(path)
	if err != nil {
		return nil, err
	}
	// sql.Open is lazy: the database file is not created until the first
	// connection is used. Ping materialises it so the read-only open below
	// has a file to open.
	if err := rw.Ping(); err != nil {
		rw.Close()
		return nil, fmt.Errorf("open: %w", err)
	}
	ro, err := OpenReadOnly(path)
	if err != nil {
		rw.Close()
		return nil, err
	}
	return &Store{rw: rw, ro: ro}, nil
}

// RW returns the read-write handle. Prefer it for writes and for reads that
// must be consistent with a just-performed write.
func (s *Store) RW() *sql.DB { return s.rw }

// RO returns the read-only handle. Statements executed on it cannot modify the
// database: the handle is opened with mode=ro and writes are rejected by the
// storage engine.
func (s *Store) RO() *sql.DB { return s.ro }

// Close closes both handles. It is safe to call on a nil Store.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	var errs []error
	if err := s.rw.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.ro.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// dsn appends options that make concurrent writers safe and foreign keys
// enforced on every connection: _foreign_keys=on applies PRAGMA foreign_keys to
// each new connection (the schema's PRAGMA only covers the connection that ran
// it), so the RESTRICT/SET NULL/CASCADE rules protecting message chains and
// lineage hold regardless of which pooled connection handles a statement. Every
// write transaction begins with BEGIN IMMEDIATE, and concurrent writers wait up
// to 5s instead of failing with SQLITE_BUSY.
func dsn(path string) string {
	opts := "_txlock=immediate&_busy_timeout=5000&_foreign_keys=on"
	if strings.ContainsRune(path, '?') {
		return path + "&" + opts
	}
	return path + "?" + opts
}

// OpenReadOnly opens a read-only connection to the database at path. The
// connection is opened with SQLite's mode=ro URI flag, so writes are rejected
// by the storage engine itself; it is the handle subagent_result and other
// read paths execute against. The caller owns the handle and must close it.
// In-memory databases are not supported (a second connection to ":memory:"
// would be a separate, empty database).
func OpenReadOnly(path string) (*sql.DB, error) {
	if path == ":memory:" {
		return nil, fmt.Errorf("open read-only: :memory: is not supported; use a file path")
	}
	// mode=ro must go through the file: URI form for SQLite to honour it;
	// the plain dsn() form is parsed by the driver and would not see it.
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_txlock=deferred&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open read-only: %w", err)
	}
	return db, nil
}

func OpenDB(path string) (*sql.DB, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, err
	}
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	}

	return db, nil
}
