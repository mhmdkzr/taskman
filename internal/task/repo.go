package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/usage"
)

// timeLayout is the on-disk timestamp format for this table — RFC3339Nano,
// deliberately not the sessions table's Go time.String() convention (see
// schema.sql for why).
const timeLayout = time.RFC3339Nano

// NewTaskID generates a fresh task id. Like session ids, it is a UUIDv7 whose
// embedded timestamp doubles as the task's created_at.
func NewTaskID() uuid.UUID {
	return uuid.NewV7()
}

// CreateTask inserts a new task in TaskStatusCreated — the only status a task
// can be created in; any Status the caller set on t is overwritten, so a task
// can never be persisted into created/started/completed/reviewed except by
// going through CreateTask and then StartTask/CompleteTask/ReviewTask in
// order. If t.CreatedAt is zero, it is derived from t.ID's embedded UUIDv7
// timestamp (like sessions' created_at is derived from the session id)
// rather than left for the caller to set separately. Callers are responsible
// for calling t.Validate() first if they want the enum/required-field checks
// enforced; CreateTask itself just persists what it is given.
func CreateTask(ctx context.Context, db *sql.DB, t Task) error {
	t.Status = TaskStatusCreated
	if t.CreatedAt.IsZero() {
		t.CreatedAt = timestampFromTaskID(t.ID)
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO tasks (
			id, title, status, task_type, tags, dependencies, what, "where", why, how, invariants,
			urgency, importance, risk, packages, completed_when,
			commit_type, commit_message, commit_hash, model, variant,
			session_id, review_session_id,
			tokens_input, tokens_output, tokens_total, tokens_reasoning, tokens_cache_read, tokens_cache_write,
			created_at, started_at, completed_at, reviewed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		insertArgs(t)...)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

// StartTask transitions a task from TaskStatusCreated to TaskStatusStarted:
// it records the execution session and stamps started_at. It fails if the
// task is not currently TaskStatusCreated (already started, or does not
// exist), so a task can't be started twice or skip being created.
func StartTask(ctx context.Context, db *sql.DB, id, sessionID uuid.UUID) error {
	res, err := db.ExecContext(ctx, `
		UPDATE tasks SET status = ?, session_id = ?, started_at = ?
		WHERE id = ? AND status = ?`,
		string(TaskStatusStarted), sessionID.String(), nullTime(time.Now()),
		id.String(), string(TaskStatusCreated))
	if err != nil {
		return fmt.Errorf("start task %s: %w", id, err)
	}
	return requireTransition(ctx, db, id, res, TaskStatusCreated, "start")
}

// CompleteTask transitions a task from TaskStatusStarted to
// TaskStatusCompleted: it records the executor's proposed commit info and
// its execution token usage, and stamps completed_at. It fails if the task
// is not currently TaskStatusStarted.
func CompleteTask(ctx context.Context, db *sql.DB, id uuid.UUID, commitType, commitMessage string, execUsage usage.TokenUsage) error {
	res, err := db.ExecContext(ctx, `
		UPDATE tasks SET
			status = ?, commit_type = ?, commit_message = ?, completed_at = ?,
			tokens_input = ?, tokens_output = ?, tokens_total = ?,
			tokens_reasoning = ?, tokens_cache_read = ?, tokens_cache_write = ?
		WHERE id = ? AND status = ?`,
		string(TaskStatusCompleted), nullStr(commitType), nullStr(commitMessage), nullTime(time.Now()),
		nullUsageField(execUsage, execUsage.InputTokens), nullUsageField(execUsage, execUsage.OutputTokens),
		nullUsageField(execUsage, execUsage.TotalTokens), nullUsageField(execUsage, execUsage.ReasoningTokens),
		nullUsageField(execUsage, execUsage.CacheReadTokens), nullUsageField(execUsage, execUsage.CacheWriteTokens),
		id.String(), string(TaskStatusStarted))
	if err != nil {
		return fmt.Errorf("complete task %s: %w", id, err)
	}
	return requireTransition(ctx, db, id, res, TaskStatusStarted, "complete")
}

// ReviewTask transitions a task from TaskStatusCompleted to
// TaskStatusReviewed: it records the review session, the actual commit hash
// and the commit agent's final commit message (once the orchestrator has
// committed the reviewed result), adds the review's token usage to the
// execution usage already recorded, and stamps reviewed_at. It fails if the
// task is not currently TaskStatusCompleted.
func ReviewTask(ctx context.Context, db *sql.DB, id, reviewSessionID uuid.UUID, commitHash, commitMessage string, reviewUsage usage.TokenUsage) error {
	commitType, subject := splitCommitMessage(commitMessage)
	res, err := db.ExecContext(ctx, `
		UPDATE tasks SET
			status = ?, review_session_id = ?, commit_hash = ?, commit_type = ?, commit_message = ?, reviewed_at = ?,
			tokens_input       = COALESCE(tokens_input, 0)       + ?,
			tokens_output      = COALESCE(tokens_output, 0)      + ?,
			tokens_total       = COALESCE(tokens_total, 0)       + ?,
			tokens_reasoning   = COALESCE(tokens_reasoning, 0)   + ?,
			tokens_cache_read  = COALESCE(tokens_cache_read, 0)  + ?,
			tokens_cache_write = COALESCE(tokens_cache_write, 0) + ?
		WHERE id = ? AND status = ?`,
		string(TaskStatusReviewed), reviewSessionID.String(), nullStr(commitHash), nullStr(commitType), nullStr(subject), nullTime(time.Now()),
		reviewUsage.InputTokens, reviewUsage.OutputTokens, reviewUsage.TotalTokens,
		reviewUsage.ReasoningTokens, reviewUsage.CacheReadTokens, reviewUsage.CacheWriteTokens,
		id.String(), string(TaskStatusCompleted))
	if err != nil {
		return fmt.Errorf("review task %s: %w", id, err)
	}
	return requireTransition(ctx, db, id, res, TaskStatusCompleted, "review")
}

// splitCommitMessage splits a conventional-commit message like
// "feat: add X" into its type and subject ("feat", "add X"). When the message
// has no type prefix the type is "" and the whole message is the subject.
func splitCommitMessage(msg string) (commitType, subject string) {
	msg = strings.TrimSpace(msg)
	idx := strings.Index(msg, ":")
	if idx <= 0 {
		return "", msg
	}
	return strings.TrimSpace(msg[:idx]), strings.TrimSpace(msg[idx+1:])
}

// ResetTask returns a task to TaskStatusCreated, clearing every field
// StartTask/CompleteTask/ReviewTask set: both session ids, commit info,
// token usage, and the started/completed/reviewed timestamps. Use it to
// recover a task a pipeline run left stuck mid-lifecycle — e.g. the process
// died between StartTask and CompleteTask, or the review/land phase failed
// after CompleteTask — since there is otherwise no way back to created once
// a task has started, and taskman run refuses to touch a task that isn't
// TaskStatusCreated. It fails if the task is already TaskStatusCreated
// (nothing to reset) or does not exist.
func ResetTask(ctx context.Context, db *sql.DB, id uuid.UUID) error {
	res, err := db.ExecContext(ctx, `
		UPDATE tasks SET
			status = ?, session_id = NULL, review_session_id = NULL,
			commit_type = NULL, commit_message = NULL, commit_hash = NULL,
			started_at = NULL, completed_at = NULL, reviewed_at = NULL,
			tokens_input = NULL, tokens_output = NULL, tokens_total = NULL,
			tokens_reasoning = NULL, tokens_cache_read = NULL, tokens_cache_write = NULL
		WHERE id = ? AND status != ?`,
		string(TaskStatusCreated), id.String(), string(TaskStatusCreated))
	if err != nil {
		return fmt.Errorf("reset task %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("reset task %s: %w", id, err)
	}
	if n > 0 {
		return nil
	}
	if _, getErr := GetTask(ctx, db, id); getErr != nil {
		return fmt.Errorf("reset task %s: not found", id)
	}
	return fmt.Errorf("reset task %s: invalid transition — already %s", id, TaskStatusCreated)
}

// requireTransition turns a zero-rows-affected guarded UPDATE into a precise
// error: not found, or found but not in the expected fromStatus (the actual
// invalid-transition case this package exists to prevent).
func requireTransition(ctx context.Context, db *sql.DB, id uuid.UUID, res sql.Result, fromStatus TaskStatus, verb string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s task %s: %w", verb, id, err)
	}
	if n > 0 {
		return nil
	}
	existing, getErr := GetTask(ctx, db, id)
	if getErr != nil {
		return fmt.Errorf("%s task %s: not found", verb, id)
	}
	return fmt.Errorf("%s task %s: invalid transition — status is %s, want %s", verb, id, existing.Status, fromStatus)
}

// EditTask updates a task's descriptive spec (title, task type, tags,
// dependencies, what/where/why/how, invariants, urgency/importance/risk,
// packages, completed_when) — never its status, session, commit, or
// timestamp columns; those only change through
// StartTask/CompleteTask/ReviewTask. It only applies to a task still in
// TaskStatusCreated: once a task has started, changing its spec out from
// under the running execution would be unsafe, so this fails with the same
// "invalid transition" shape as the lifecycle functions.
func EditTask(ctx context.Context, db *sql.DB, t Task) error {
	res, err := db.ExecContext(ctx, `
		UPDATE tasks SET
			title = ?, task_type = ?, tags = ?, dependencies = ?, what = ?, "where" = ?, why = ?, how = ?, invariants = ?,
			urgency = ?, importance = ?, risk = ?, packages = ?, completed_when = ?
		WHERE id = ? AND status = ?`,
		t.Title, t.TaskType, marshalJSON(t.Tags), marshalJSON(t.Dependencies),
		t.What, marshalJSON(t.Where), t.Why, t.How, marshalJSON(t.Invariants),
		t.Urgency, t.Importance, nullStr(t.Risk), marshalJSON(t.Packages), marshalJSON(t.CompletedWhen),
		t.ID.String(), string(TaskStatusCreated))
	if err != nil {
		return fmt.Errorf("edit task %s: %w", t.ID, err)
	}
	return requireTransition(ctx, db, t.ID, res, TaskStatusCreated, "edit")
}

// GetTask loads one task by id.
func GetTask(ctx context.Context, db *sql.DB, id uuid.UUID) (*Task, error) {
	row := db.QueryRowContext(ctx, taskSelect+` WHERE id = ?`, id.String())
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get task %s: %w", id, err)
	}
	return &t, nil
}

// ListTasks returns tasks newest first, optionally restricted to one status
// (pass "" for every status).
func ListTasks(ctx context.Context, db *sql.DB, status TaskStatus) ([]Task, error) {
	query := taskSelect
	var args []any
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, string(status))
	}
	query += ` ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return out, nil
}

// TaskFilter narrows SearchTasks. The zero value matches every task. Package
// and Tag match a task whose Packages/Tags list contains that exact value;
// Query is a substring match (case-insensitive for ASCII, per SQLite's
// default LIKE collation) against title/what/why/how.
type TaskFilter struct {
	Status   TaskStatus
	TaskType string
	Package  string
	Tag      string
	Query    string
}

// SearchTasks returns tasks matching every set field of f, newest first.
func SearchTasks(ctx context.Context, db *sql.DB, f TaskFilter) ([]Task, error) {
	query := taskSelect
	var where []string
	var args []any
	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, string(f.Status))
	}
	if f.TaskType != "" {
		where = append(where, "task_type = ?")
		args = append(args, f.TaskType)
	}
	if f.Package != "" {
		where = append(where, `packages LIKE ?`)
		args = append(args, "%"+jsonElementLike(f.Package)+"%")
	}
	if f.Tag != "" {
		where = append(where, `tags LIKE ?`)
		args = append(args, "%"+jsonElementLike(f.Tag)+"%")
	}
	if f.Query != "" {
		where = append(where, `(title LIKE ? OR what LIKE ? OR why LIKE ? OR how LIKE ?)`)
		q := "%" + f.Query + "%"
		args = append(args, q, q, q, q)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("search tasks: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search tasks: %w", err)
	}
	return out, nil
}

// jsonElementLike renders v as a quoted JSON string; wrapping the result in
// "%...%" for a LIKE match anchors to a whole element of a
// marshalJSON-produced array (the quotes on both sides mean it can't match
// inside a longer element's text, e.g. searching "b" won't match "bc") without
// needing real JSON-aware querying.
func jsonElementLike(v string) string {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	return string(b)
}

// DeleteTask removes a task by id.
func DeleteTask(ctx context.Context, db *sql.DB, id uuid.UUID) error {
	res, err := db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("delete task %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete task %s: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("delete task %s: not found", id)
	}
	return nil
}

const taskSelect = `
	SELECT
		id, title, status, task_type, tags, dependencies, what, "where", why, how, invariants,
		urgency, importance, risk, packages, completed_when,
		commit_type, commit_message, commit_hash, model, variant,
		session_id, review_session_id,
		tokens_input, tokens_output, tokens_total, tokens_reasoning, tokens_cache_read, tokens_cache_write,
		created_at, started_at, completed_at, reviewed_at
	FROM tasks`

// insertArgs renders a Task in exactly the column order taskSelect/CreateTask
// use, so scanTask and the writers stay in lockstep by construction.
func insertArgs(t Task) []any {
	return []any{
		t.ID.String(), t.Title, string(t.Status), t.TaskType,
		marshalJSON(t.Tags), marshalJSON(t.Dependencies),
		t.What, marshalJSON(t.Where), t.Why, t.How, marshalJSON(t.Invariants),
		t.Urgency, t.Importance, nullStr(t.Risk),
		marshalJSON(t.Packages), marshalJSON(t.CompletedWhen),
		nullStr(t.CommitType), nullStr(t.CommitMessage), nullStr(t.CommitHash),
		nullStr(t.Model), nullStr(t.Variant),
		nullUUID(t.SessionID), nullUUID(t.ReviewSessionID),
		nullUsageField(t.TokenUsage, t.TokenUsage.InputTokens),
		nullUsageField(t.TokenUsage, t.TokenUsage.OutputTokens),
		nullUsageField(t.TokenUsage, t.TokenUsage.TotalTokens),
		nullUsageField(t.TokenUsage, t.TokenUsage.ReasoningTokens),
		nullUsageField(t.TokenUsage, t.TokenUsage.CacheReadTokens),
		nullUsageField(t.TokenUsage, t.TokenUsage.CacheWriteTokens),
		nullTime(t.CreatedAt), nullTime(t.StartedAt), nullTime(t.CompletedAt), nullTime(t.ReviewedAt),
	}
}

// rowScanner is the subset of *sql.Row/*sql.Rows scanTask needs.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner) (Task, error) {
	var t Task
	var id, tags, dependencies, where, invariants, packages, completedWhen string
	var risk, commitType, commitMessage, commitHash, model, variant sql.NullString
	var sessionID, reviewSessionID sql.NullString
	var tokIn, tokOut, tokTotal, tokReason, tokCacheRead, tokCacheWrite sql.NullInt64
	var createdAt string
	var startedAt, completedAt, reviewedAt sql.NullString

	if err := row.Scan(
		&id, &t.Title, &t.Status, &t.TaskType, &tags, &dependencies, &t.What, &where, &t.Why, &t.How, &invariants,
		&t.Urgency, &t.Importance, &risk, &packages, &completedWhen,
		&commitType, &commitMessage, &commitHash, &model, &variant,
		&sessionID, &reviewSessionID,
		&tokIn, &tokOut, &tokTotal, &tokReason, &tokCacheRead, &tokCacheWrite,
		&createdAt, &startedAt, &completedAt, &reviewedAt,
	); err != nil {
		return Task{}, err
	}

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return Task{}, fmt.Errorf("parse task id %q: %w", id, err)
	}
	t.ID = parsedID

	t.Tags = unmarshalJSON(tags)
	t.Dependencies = unmarshalJSON(dependencies)
	t.Where = unmarshalJSON(where)
	t.Invariants = unmarshalJSON(invariants)
	t.Packages = unmarshalJSON(packages)
	t.CompletedWhen = unmarshalJSON(completedWhen)

	t.Risk = risk.String
	t.CommitType = commitType.String
	t.CommitMessage = commitMessage.String
	t.CommitHash = commitHash.String
	t.Model = model.String
	t.Variant = variant.String

	if t.SessionID, err = parseNullUUID(sessionID); err != nil {
		return Task{}, fmt.Errorf("parse session_id: %w", err)
	}
	if t.ReviewSessionID, err = parseNullUUID(reviewSessionID); err != nil {
		return Task{}, fmt.Errorf("parse review_session_id: %w", err)
	}

	if tokIn.Valid || tokOut.Valid || tokTotal.Valid || tokReason.Valid || tokCacheRead.Valid || tokCacheWrite.Valid {
		t.TokenUsage = usage.TokenUsage{
			InputTokens:      int(tokIn.Int64),
			OutputTokens:     int(tokOut.Int64),
			TotalTokens:      int(tokTotal.Int64),
			ReasoningTokens:  int(tokReason.Int64),
			CacheReadTokens:  int(tokCacheRead.Int64),
			CacheWriteTokens: int(tokCacheWrite.Int64),
		}
	}

	if t.CreatedAt, err = time.Parse(timeLayout, createdAt); err != nil {
		return Task{}, fmt.Errorf("parse created_at: %w", err)
	}
	if t.StartedAt, err = parseNullTime(startedAt); err != nil {
		return Task{}, fmt.Errorf("parse started_at: %w", err)
	}
	if t.CompletedAt, err = parseNullTime(completedAt); err != nil {
		return Task{}, fmt.Errorf("parse completed_at: %w", err)
	}
	if t.ReviewedAt, err = parseNullTime(reviewedAt); err != nil {
		return Task{}, fmt.Errorf("parse reviewed_at: %w", err)
	}

	return t, nil
}

func marshalJSON(v []string) string {
	if len(v) == 0 {
		return "[]"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func unmarshalJSON(s string) []string {
	if s == "" {
		return nil
	}
	var v []string
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil
	}
	return v
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullUUID(u uuid.UUID) any {
	if u == uuid.Nil() {
		return nil
	}
	return u.String()
}

func parseNullUUID(s sql.NullString) (uuid.UUID, error) {
	if !s.Valid || s.String == "" {
		return uuid.Nil(), nil
	}
	return uuid.Parse(s.String)
}

// nullUsageField stores NULL for every usage column when u is the zero value
// (no usage recorded yet), matching messages.* in schema.sql: a row of NULLs
// means "not recorded", not "recorded as zero".
func nullUsageField(u usage.TokenUsage, field int) any {
	if u == (usage.TokenUsage{}) {
		return nil
	}
	return field
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(timeLayout)
}

func parseNullTime(s sql.NullString) (time.Time, error) {
	if !s.Valid || s.String == "" {
		return time.Time{}, nil
	}
	return time.Parse(timeLayout, s.String)
}

// timestampFromTaskID decodes the UTC time a UUIDv7 embeds in its first 48
// bits (big-endian) — the same layout store.unixMillisFromUUIDV7 decodes for
// session ids, just operating on the stdlib uuid.UUID array directly.
func timestampFromTaskID(id uuid.UUID) time.Time {
	ms := int64(id[0])<<40 | int64(id[1])<<32 | int64(id[2])<<24 |
		int64(id[3])<<16 | int64(id[4])<<8 | int64(id[5])
	return time.UnixMilli(ms).UTC()
}
