package task

import (
	"context"
	"database/sql"
	"encoding/json"
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
			model, reasoning_effort, commit_hash, branch, failure_reason, pipeline_step, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID.String(), t.Definition, t.Specification, t.State,
		t.Importance, t.Urgency, t.Complexity, t.Effort, t.Risk, t.Autonomy,
		t.Model, t.ReasoningEffort, t.CommitHash, t.Branch, t.FailureReason, t.PipelineStep, now); err != nil {
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
	var reasoningEffort, branch, failureReason sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT id, definition, specification, state,
			importance, urgency, complexity, effort, risk, autonomy,
			model, reasoning_effort, commit_hash, branch, failure_reason, pipeline_step
		FROM tasks
		WHERE id = ? AND deleted_at IS NULL`, id.String()).Scan(
		&idString, &t.Definition, &t.Specification, &t.State,
		&t.Importance, &t.Urgency, &t.Complexity, &t.Effort, &t.Risk, &t.Autonomy,
		&t.Model, &reasoningEffort, &t.CommitHash, &branch, &failureReason, &t.PipelineStep,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrTaskNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("query task: %w", err)
	}
	t.ReasoningEffort = reasoningEffort.String
	t.Branch = branch.String
	t.FailureReason = failureReason.String
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
	addValuesFilter("reasoning_effort", filter.ReasoningEfforts)
	addValuesFilter("commit_hash", filter.CommitHashes)
	addValuesFilter("branch", filter.Branches)
	query.WriteString(" ORDER BY created_at, id")

	rows, err := db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer closeTaskRows(rows)

	var ids []uuid.UUID
	for rows.Next() {
		var idString string
		if err := rows.Scan(&idString); err != nil {
			return nil, fmt.Errorf("scan task id: %w", err)
		}
		id, err := uuid.Parse(idString)
		if err != nil {
			return nil, fmt.Errorf("parse task id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
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

// LinkSession records that sessionID was dispatched for taskID. Callers
// that dispatch a session on a task's behalf (e.g. internal/pipeline) must
// call this immediately after creating the session and before running it,
// so the link survives even if the session's own turn crashes mid-run.
func LinkSession(ctx context.Context, db *sql.DB, taskID, sessionID uuid.UUID) error {
	if _, err := db.ExecContext(ctx,
		`INSERT INTO tasks_sessions (task_id, session_id) VALUES (?, ?)`,
		taskID.String(), sessionID.String()); err != nil {
		return fmt.Errorf("link task session: %w", err)
	}
	return nil
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
			model = ?, reasoning_effort = ?, commit_hash = ?, branch = ?, failure_reason = ?, pipeline_step = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		t.Definition, t.Specification, t.State,
		t.Importance, t.Urgency, t.Complexity, t.Effort, t.Risk, t.Autonomy,
		t.Model, t.ReasoningEffort, t.CommitHash, t.Branch, t.FailureReason, t.PipelineStep, time.Now().UTC().Format(time.RFC3339Nano), t.ID.String())
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

// InsertReviewResult records one review attempt's outcome for a task,
// assigning it the next attempt number for that task (1 for the first
// review). Called the moment a review finishes, before any later stage
// (fix, commit, ...) runs, so a crash afterward never loses what the
// review found.
func InsertReviewResult(ctx context.Context, db *sql.DB, r ReviewResult) (ReviewResult, error) {
	if r.TaskID == (uuid.UUID{}) {
		return ReviewResult{}, fmt.Errorf("insert review result: task id is required")
	}
	findings, err := json.Marshal(r.Findings)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("marshal review findings: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("begin insert review result: %w", err)
	}
	defer rollbackTaskTx(tx)

	var maxAttempt sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(attempt) FROM task_review_results WHERE task_id = ?`, r.TaskID.String(),
	).Scan(&maxAttempt); err != nil {
		return ReviewResult{}, fmt.Errorf("resolve next review attempt: %w", err)
	}
	r.Attempt = int(maxAttempt.Int64) + 1
	r.ID = uuid.NewV7()

	sessionID := sql.NullString{String: r.SessionID.String(), Valid: r.SessionID != (uuid.UUID{})}
	approved := 0
	if r.Approved {
		approved = 1
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO task_review_results (id, task_id, session_id, attempt, approved, findings, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID.String(), r.TaskID.String(), sessionID, r.Attempt, approved, string(findings),
		time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return ReviewResult{}, fmt.Errorf("insert review result: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ReviewResult{}, fmt.Errorf("commit insert review result: %w", err)
	}
	return r, nil
}

// ReviewResultsForTask returns every review attempt recorded for taskID,
// oldest attempt first.
func ReviewResultsForTask(ctx context.Context, db *sql.DB, taskID uuid.UUID) ([]ReviewResult, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, session_id, attempt, approved, findings
		FROM task_review_results
		WHERE task_id = ?
		ORDER BY attempt`, taskID.String())
	if err != nil {
		return nil, fmt.Errorf("query review results: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close review result rows", "error", err)
		}
	}()

	var results []ReviewResult
	for rows.Next() {
		var idString string
		var sessionIDString sql.NullString
		var approved int
		var findings string
		r := ReviewResult{TaskID: taskID}
		if err := rows.Scan(&idString, &sessionIDString, &r.Attempt, &approved, &findings); err != nil {
			return nil, fmt.Errorf("scan review result: %w", err)
		}
		if r.ID, err = uuid.Parse(idString); err != nil {
			return nil, fmt.Errorf("parse review result id: %w", err)
		}
		if sessionIDString.Valid {
			if r.SessionID, err = uuid.Parse(sessionIDString.String); err != nil {
				return nil, fmt.Errorf("parse review result session id: %w", err)
			}
		}
		r.Approved = approved != 0
		if err := json.Unmarshal([]byte(findings), &r.Findings); err != nil {
			return nil, fmt.Errorf("unmarshal review findings: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate review results: %w", err)
	}
	return results, nil
}

func closeTaskRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		slog.Error("close task rows", "error", err)
	}
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
