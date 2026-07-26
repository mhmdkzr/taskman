package pg

import (
	"database/sql"
	"errors"
	"log/slog"
)

// CloseRows closes SQL rows for deferred cleanup paths.
func CloseRows(rows *sql.Rows) {
	if rows == nil {
		return
	}
	if err := rows.Close(); err != nil {
		slog.Error("failed to close sql rows", "error", err)
	}
}

// CloseDB closes a SQL database handle for deferred cleanup paths.
func CloseDB(db *sql.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		slog.Error("failed to close sql database", "error", err)
	}
}

// RollbackTx rolls back a SQL transaction for deferred cleanup paths.
func RollbackTx(tx *sql.Tx) {
	if tx == nil {
		return
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		slog.Error("failed to rollback sql transaction", "error", err)
	}
}
