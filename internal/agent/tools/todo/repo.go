package todo

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
	"github.com/mhmdkzr/loop/internal/store"
)

var errSessionNotFound = errors.New("session not found")

// ReplaceTodos replaces the active session's todo list atomically.
func ReplaceTodos(ctx context.Context, st *store.Store, id sessions.SessionID, todos []Todo) error {
	for _, todo := range todos {
		if strings.TrimSpace(todo.Content) == "" {
			return fmt.Errorf("%w: content is required", ErrInvalidTodo)
		}
		if !validTodoStatus(todo.Status) {
			return fmt.Errorf("%w: invalid status %q", ErrInvalidTodo, todo.Status)
		}
		if !validTodoPriority(todo.Priority) {
			return fmt.Errorf("%w: invalid priority %q", ErrInvalidTodo, todo.Priority)
		}
	}

	tx, err := st.RW().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin todo transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("rollback todo transaction", "error", err)
		}
	}()

	result, err := tx.ExecContext(ctx, `
		DELETE FROM session_todos
		WHERE session_id = ? AND EXISTS (
			SELECT 1 FROM agent_sessions WHERE session_id = ? AND deleted_at IS NULL
		)`, id.String(), id.String())
	if err != nil {
		return fmt.Errorf("delete session todos: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("check session todos: %w", err)
	} else if affected == 0 {
		var exists int
		if err := tx.QueryRowContext(ctx,
			`SELECT 1 FROM agent_sessions WHERE session_id = ? AND deleted_at IS NULL`, id.String(),
		).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return errSessionNotFound
		} else if err != nil {
			return fmt.Errorf("check session: %w", err)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for position, todo := range todos {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO session_todos (todo_id, session_id, content, status, priority, position, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewV7().String(), id.String(), todo.Content, todo.Status, todo.Priority, position, now, now); err != nil {
			return fmt.Errorf("insert session todo: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit session todos: %w", err)
	}
	return nil
}

func validTodoStatus(status TodoStatus) bool {
	return status == TodoPending || status == TodoInProgress || status == TodoCompleted || status == TodoCancelled
}

func validTodoPriority(priority TodoPriority) bool {
	return priority == TodoHigh || priority == TodoMedium || priority == TodoLow
}
