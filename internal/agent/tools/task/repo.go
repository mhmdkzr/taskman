package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"uuid"
)

var ErrTaskNotFound = errors.New("task not found")

func CreateTask(ctx context.Context, db *sql.DB, t Task) error {
	if t.ID == (uuid.UUID{}) {
		return fmt.Errorf("create task: id is required")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create task: %w", err)
	}
	defer rollbackTaskTx(tx)

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tasks (
			id, definition, specification, state,
			importance, urgency, complexity, effort, risk, autonomy,
			model, commit_hash, branch, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID.String(), t.Definition, t.Specification, t.State,
		t.Importance, t.Urgency, t.Complexity, t.Effort, t.Risk, t.Autonomy,
		t.Model, t.CommitHash, t.Branch, now); err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	if err := replaceTaskLabels(ctx, tx, t.ID, t.Labels); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create task: %w", err)
	}
	return nil
}

func GetTask(ctx context.Context, db *sql.DB, id uuid.UUID) (Task, error) {
	var t Task
	var idString string
	var branch sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT id, definition, specification, state,
			importance, urgency, complexity, effort, risk, autonomy,
			model, commit_hash, branch
		FROM tasks
		WHERE id = ? AND deleted_at IS NULL`, id.String()).Scan(
		&idString, &t.Definition, &t.Specification, &t.State,
		&t.Importance, &t.Urgency, &t.Complexity, &t.Effort, &t.Risk, &t.Autonomy,
		&t.Model, &t.CommitHash, &branch,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrTaskNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("query task: %w", err)
	}
	t.Branch = branch.String
	t.ID, err = uuid.Parse(idString)
	if err != nil {
		return Task{}, fmt.Errorf("parse task id: %w", err)
	}
	labels, err := taskLabels(ctx, db, t.ID)
	if err != nil {
		return Task{}, err
	}
	t.Labels = labels
	return t, nil
}

func ListTasks(ctx context.Context, db *sql.DB, filter TaskFilter) ([]Task, error) {
	query := strings.Builder{}
	query.WriteString(`SELECT id FROM tasks WHERE deleted_at IS NULL`)
	args := make([]any, 0)

	addValuesFilter := func(column string, values []string) {
		if len(values) == 0 {
			return
		}
		query.WriteString(" AND ")
		query.WriteString(column)
		query.WriteString(" IN (")
		query.WriteString(placeholders(len(values)))
		query.WriteString(")")
		for _, value := range values {
			args = append(args, value)
		}
	}
	addIntegerValuesFilter := func(column string, values []int64) {
		if len(values) == 0 {
			return
		}
		query.WriteString(" AND ")
		query.WriteString(column)
		query.WriteString(" IN (")
		query.WriteString(placeholders(len(values)))
		query.WriteString(")")
		for _, value := range values {
			args = append(args, value)
		}
	}
	addUUIDFilter := func(column string, values []uuid.UUID) {
		valuesAsStrings := make([]string, len(values))
		for i, value := range values {
			valuesAsStrings[i] = value.String()
		}
		addValuesFilter(column, valuesAsStrings)
	}
	addLabelFilter := func(values []string) {
		if len(values) == 0 {
			return
		}
		query.WriteString(` AND EXISTS (
			SELECT 1 FROM tasks_labels
			WHERE task_id = tasks.id AND label IN (`)
		query.WriteString(placeholders(len(values)))
		query.WriteString(`))`)
		for _, value := range values {
			args = append(args, value)
		}
	}

	addUUIDFilter("id", filter.IDs)
	addValuesFilter("state", taskStates(filter.State))
	addLabelFilter(filter.Labels)
	addIntegerValuesFilter("importance", levels(filter.Importance))
	addIntegerValuesFilter("urgency", levels(filter.Urgency))
	addIntegerValuesFilter("complexity", levels(filter.Complexity))
	addIntegerValuesFilter("effort", levels(filter.Effort))
	addIntegerValuesFilter("risk", levels(filter.Risk))
	addIntegerValuesFilter("autonomy", levels(filter.Autonomy))
	addValuesFilter("model", filter.Model)
	addValuesFilter("commit_hash", filter.CommitHashes)
	addValuesFilter("branch", filter.Branches)
	query.WriteString(" ORDER BY created_at, id")

	rows, err := db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var idString string
		if err := rows.Scan(&idString); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan task id: %w", err)
		}
		id, err := uuid.Parse(idString)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("parse task id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close task rows: %w", err)
	}

	tasks := make([]Task, 0, len(ids))
	for _, id := range ids {
		t, err := GetTask(ctx, db, id)
		if err != nil {
			return nil, fmt.Errorf("load listed task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// SessionIDsForTask returns every session linked to taskID via tasks_sessions,
// in the order they were linked.
func SessionIDsForTask(ctx context.Context, db *sql.DB, taskID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT session_id FROM tasks_sessions WHERE task_id = ?`, taskID.String())
	if err != nil {
		return nil, fmt.Errorf("query task session ids: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close task session id rows", "error", err)
		}
	}()

	var ids []uuid.UUID
	for rows.Next() {
		var idString string
		if err := rows.Scan(&idString); err != nil {
			return nil, fmt.Errorf("scan task session id: %w", err)
		}
		id, err := uuid.Parse(idString)
		if err != nil {
			return nil, fmt.Errorf("parse task session id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task session ids: %w", err)
	}
	return ids, nil
}

func UpdateTask(ctx context.Context, db *sql.DB, t Task) error {
	if t.ID == (uuid.UUID{}) {
		return fmt.Errorf("update task: id is required")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update task: %w", err)
	}
	defer rollbackTaskTx(tx)

	result, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET definition = ?, specification = ?, state = ?,
			importance = ?, urgency = ?, complexity = ?, effort = ?, risk = ?, autonomy = ?,
			model = ?, commit_hash = ?, branch = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		t.Definition, t.Specification, t.State,
		t.Importance, t.Urgency, t.Complexity, t.Effort, t.Risk, t.Autonomy,
		t.Model, t.CommitHash, t.Branch, time.Now().UTC().Format(time.RFC3339Nano), t.ID.String())
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated task: %w", err)
	}
	if affected == 0 {
		return ErrTaskNotFound
	}
	if err := replaceTaskLabels(ctx, tx, t.ID, t.Labels); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update task: %w", err)
	}
	return nil
}

func DeleteTask(ctx context.Context, db *sql.DB, id uuid.UUID) error {
	result, err := db.ExecContext(ctx, `
		UPDATE tasks SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL`, time.Now().UTC().Format(time.RFC3339Nano), id.String())
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted task: %w", err)
	}
	if affected == 0 {
		return ErrTaskNotFound
	}
	return nil
}

func replaceTaskLabels(ctx context.Context, tx *sql.Tx, id uuid.UUID, labels []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks_labels WHERE task_id = ?`, id.String()); err != nil {
		return fmt.Errorf("delete task labels: %w", err)
	}
	for _, label := range labels {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO tasks_labels (task_id, label) VALUES (?, ?)`, id.String(), label); err != nil {
			return fmt.Errorf("insert task label: %w", err)
		}
	}
	return nil
}

func taskLabels(ctx context.Context, db *sql.DB, id uuid.UUID) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT label FROM tasks_labels WHERE task_id = ? ORDER BY label`, id.String())
	if err != nil {
		return nil, fmt.Errorf("query task labels: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close task label rows", "error", err)
		}
	}()

	var labels []string
	for rows.Next() {
		var label string
		if err := rows.Scan(&label); err != nil {
			return nil, fmt.Errorf("scan task label: %w", err)
		}
		labels = append(labels, label)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task labels: %w", err)
	}
	return labels, nil
}

func rollbackTaskTx(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		slog.Error("rollback task transaction", "error", err)
	}
}

func placeholders(n int) string {
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func levels(values []Level) []int64 {
	result := make([]int64, len(values))
	for i, value := range values {
		result[i] = int64(value)
	}
	return result
}

func taskStates(values []TaskState) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}
	return result
}
