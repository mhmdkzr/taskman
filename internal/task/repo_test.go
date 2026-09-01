package task

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/usage"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := store.Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db
}

// newTestSessionID creates a real (near-empty) session row, since
// tasks.session_id/review_session_id reference sessions(id).
func newTestSessionID(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	id, err := store.NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID: %v", err)
	}
	if err := store.CreateSession(context.Background(), db, id, "test-model", "medium", 10, "", nil); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	u, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("parse session id: %v", err)
	}
	return u
}

func newTestTask() Task {
	return Task{
		ID:            NewTaskID(),
		Title:         "Fix nil pointer",
		TaskType:      "bug",
		What:          "Guard against a nil client before calling Send.",
		Why:           "Send panics when the client is nil.",
		How:           "Add a nil check at the top of Send.",
		Urgency:       "low",
		Importance:    "medium",
		Packages:      []string{"internal/foo"},
		CompletedWhen: []string{"tests pass", "vet is clean"},
		Variant:       "medium",
	}
}

// TestCreateTaskForcesCreatedStatus verifies CreateTask always persists
// TaskStatusCreated, regardless of what Status the caller set — a task can
// only enter the created/started/completed/reviewed lifecycle at its start.
func TestCreateTaskForcesCreatedStatus(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	tk := newTestTask()
	tk.Status = TaskStatusReviewed // caller tries to skip the lifecycle
	if err := CreateTask(ctx, db, tk); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	got, err := GetTask(ctx, db, tk.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != TaskStatusCreated {
		t.Errorf("status = %q, want %q", got.Status, TaskStatusCreated)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt not derived from the task id")
	}
}

// TestTaskLifecycle walks a task through the full created -> started ->
// completed -> reviewed sequence and checks each stage's fields land, usage
// accumulates across execution + review, and the row round-trips via
// GetTask/ListTasks.
func TestTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	tk := newTestTask()
	if err := CreateTask(ctx, db, tk); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	execSessionID := newTestSessionID(t, db)
	if err := StartTask(ctx, db, tk.ID, execSessionID); err != nil {
		t.Fatalf("StartTask: %v", err)
	}
	got, err := GetTask(ctx, db, tk.ID)
	if err != nil {
		t.Fatalf("GetTask after start: %v", err)
	}
	if got.Status != TaskStatusStarted || got.SessionID != execSessionID || got.StartedAt.IsZero() {
		t.Errorf("after start = %+v", got)
	}

	execUsage := usage.TokenUsage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150}
	if err := CompleteTask(ctx, db, tk.ID, "fix", "Guard Send against a nil client", execUsage); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	got, err = GetTask(ctx, db, tk.ID)
	if err != nil {
		t.Fatalf("GetTask after complete: %v", err)
	}
	if got.Status != TaskStatusCompleted || got.CommitType != "fix" || got.CompletedAt.IsZero() {
		t.Errorf("after complete = %+v", got)
	}
	if got.TokenUsage != execUsage {
		t.Errorf("token usage after complete = %+v, want %+v", got.TokenUsage, execUsage)
	}

	reviewSessionID := newTestSessionID(t, db)
	reviewUsage := usage.TokenUsage{InputTokens: 40, OutputTokens: 10, TotalTokens: 50}
	if err := ReviewTask(ctx, db, tk.ID, reviewSessionID, "abc123", reviewUsage); err != nil {
		t.Fatalf("ReviewTask: %v", err)
	}
	got, err = GetTask(ctx, db, tk.ID)
	if err != nil {
		t.Fatalf("GetTask after review: %v", err)
	}
	if got.Status != TaskStatusReviewed || got.ReviewSessionID != reviewSessionID || got.CommitHash != "abc123" || got.ReviewedAt.IsZero() {
		t.Errorf("after review = %+v", got)
	}
	wantUsage := execUsage.Add(reviewUsage)
	if got.TokenUsage != wantUsage {
		t.Errorf("token usage after review = %+v, want %+v (execution + review)", got.TokenUsage, wantUsage)
	}

	list, err := ListTasks(ctx, db, TaskStatusReviewed)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(list) != 1 || list[0].ID != tk.ID {
		t.Errorf("list = %+v", list)
	}

	if err := DeleteTask(ctx, db, tk.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if _, err := GetTask(ctx, db, tk.ID); err == nil {
		t.Error("GetTask after delete: want error")
	}
}

// TestInvalidTransitionsRejected verifies each transition function refuses to
// run when the task isn't in the expected prior status, and says so clearly.
func TestInvalidTransitionsRejected(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	tk := newTestTask()
	if err := CreateTask(ctx, db, tk); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Can't complete or review before starting.
	if err := CompleteTask(ctx, db, tk.ID, "fix", "msg", usage.TokenUsage{}); err == nil {
		t.Error("CompleteTask on a created task: want error")
	} else if !strings.Contains(err.Error(), "invalid transition") {
		t.Errorf("CompleteTask error = %q, want it to mention invalid transition", err)
	}
	if err := ReviewTask(ctx, db, tk.ID, NewTaskID(), "hash", usage.TokenUsage{}); err == nil {
		t.Error("ReviewTask on a created task: want error")
	}

	if err := StartTask(ctx, db, tk.ID, newTestSessionID(t, db)); err != nil {
		t.Fatalf("StartTask: %v", err)
	}

	// Can't start twice.
	if err := StartTask(ctx, db, tk.ID, NewTaskID()); err == nil {
		t.Error("StartTask twice: want error")
	}
	// Can't review before completing.
	if err := ReviewTask(ctx, db, tk.ID, NewTaskID(), "hash", usage.TokenUsage{}); err == nil {
		t.Error("ReviewTask on a started task: want error")
	}

	// A transition on a task that doesn't exist at all is still a clear error.
	if err := StartTask(ctx, db, NewTaskID(), NewTaskID()); err == nil {
		t.Error("StartTask on a nonexistent task: want error")
	}
}

// TestEditTask verifies EditTask updates the descriptive spec while it's
// still TaskStatusCreated, and is rejected once the task has started.
func TestEditTask(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	tk := newTestTask()
	if err := CreateTask(ctx, db, tk); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	got, err := GetTask(ctx, db, tk.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	got.Title = "Fix nil pointer in Send"
	got.Packages = []string{"internal/foo", "internal/bar"}
	got.Risk = "high"
	if err := EditTask(ctx, db, *got); err != nil {
		t.Fatalf("EditTask: %v", err)
	}

	got, err = GetTask(ctx, db, tk.ID)
	if err != nil {
		t.Fatalf("GetTask after edit: %v", err)
	}
	if got.Title != "Fix nil pointer in Send" || len(got.Packages) != 2 || got.Risk != "high" {
		t.Errorf("after edit = %+v", got)
	}

	if err := StartTask(ctx, db, tk.ID, newTestSessionID(t, db)); err != nil {
		t.Fatalf("StartTask: %v", err)
	}
	got.Title = "should not apply"
	if err := EditTask(ctx, db, *got); err == nil {
		t.Error("EditTask on a started task: want error")
	} else if !strings.Contains(err.Error(), "invalid transition") {
		t.Errorf("EditTask error = %q, want it to mention invalid transition", err)
	}
}

// TestSearchTasks verifies each TaskFilter field narrows results, and that
// package/tag matching anchors to a whole element rather than a substring.
func TestSearchTasks(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)

	a := newTestTask()
	a.Title = "Fix nil pointer"
	a.TaskType = "bug"
	a.Packages = []string{"internal/foo"}
	a.Tags = []string{"nil-safety"}
	if err := CreateTask(ctx, db, a); err != nil {
		t.Fatalf("CreateTask a: %v", err)
	}

	b := newTestTask()
	b.Title = "Add missing test"
	b.TaskType = "test"
	b.Packages = []string{"internal/foobar"} // shares "foo" as a substring, not as an element
	b.Tags = []string{"coverage"}
	if err := CreateTask(ctx, db, b); err != nil {
		t.Fatalf("CreateTask b: %v", err)
	}

	byType, err := SearchTasks(ctx, db, TaskFilter{TaskType: "bug"})
	if err != nil || len(byType) != 1 || byType[0].ID != a.ID {
		t.Errorf("SearchTasks by type = %+v, err=%v", byType, err)
	}

	byPackage, err := SearchTasks(ctx, db, TaskFilter{Package: "internal/foo"})
	if err != nil || len(byPackage) != 1 || byPackage[0].ID != a.ID {
		t.Errorf("SearchTasks by package = %+v, err=%v (must not also match internal/foobar)", byPackage, err)
	}

	byQuery, err := SearchTasks(ctx, db, TaskFilter{Query: "missing"})
	if err != nil || len(byQuery) != 1 || byQuery[0].ID != b.ID {
		t.Errorf("SearchTasks by query = %+v, err=%v", byQuery, err)
	}

	all, err := SearchTasks(ctx, db, TaskFilter{})
	if err != nil || len(all) != 2 {
		t.Errorf("SearchTasks with no filter = %+v, err=%v", all, err)
	}
}
