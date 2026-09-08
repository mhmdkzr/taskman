package task

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/loop/internal/store"
	"github.com/mhmdkzr/loop/migrations"
	"github.com/mhmdkzr/loop/pkg/migrate"
)

// seedSession inserts the minimal provider/model/prompt/agent/session rows
// an agent_sessions row needs to satisfy foreign keys, and returns its ID.
func seedSession(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	providerID := uuid.NewV7()
	if _, err := db.Exec(`INSERT INTO model_providers (provider_id, provider_name) VALUES (?, ?)`,
		providerID.String(), "test-provider-"+providerID.String()); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	modelID := uuid.NewV7()
	modelQuery := `INSERT INTO models (model_id, provider_id, model_name, context_window, has_vision)
		VALUES (?, ?, ?, 0, 0)`
	if _, err := db.Exec(modelQuery, modelID.String(), providerID.String(), "test-model"); err != nil {
		t.Fatalf("insert model: %v", err)
	}
	promptID := uuid.NewV7()
	promptQuery := `INSERT INTO prompt_templates (prompt_id, prompt_name, template_body, params_schema, version)
		VALUES (?, ?, ?, '{}', 1)`
	if _, err := db.Exec(
		promptQuery, promptID.String(), "test-prompt-"+promptID.String(), "you are a test agent"); err != nil {
		t.Fatalf("insert prompt template: %v", err)
	}
	agentID := uuid.NewV7()
	agentQuery := `INSERT INTO agents (agent_id, agent_name, prompt_id, model_id, created_at)
		VALUES (?, ?, ?, ?, ?)`
	if _, err := db.Exec(agentQuery,
		agentID.String(), "test-agent-"+agentID.String(), promptID.String(), modelID.String(), now); err != nil {
		t.Fatalf("insert agent: %v", err)
	}
	sessionID := uuid.NewV7()
	sessionQuery := `INSERT INTO agent_sessions (session_id, agent_id, model_id, system_prompt, created_at)
		VALUES (?, ?, ?, ?, ?)`
	if _, err := db.Exec(sessionQuery, sessionID.String(), agentID.String(), modelID.String(), "sys", now); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return sessionID
}

func openTaskTestDB(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "tasks.sqlite"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := migrate.Migrate(t.Context(), st.RW(), migrations.GetMigrationsFS()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return st
}

func testTask() Task {
	return Task{
		ID:            uuid.NewV7(),
		Definition:    "Implement task repository",
		Specification: "Persist task fields and labels",
		State:         TaskStateCreated,
		Labels:        []string{"backend", "urgent"},
		Importance:    LevelHigh,
		Urgency:       LevelMedium,
		Complexity:    LevelLow,
		Effort:        LevelMedium,
		Risk:          LevelLow,
		Autonomy:      LevelHigh,
		Model:         "gpt-test",
		CommitHash:    "abc123",
	}
}

func TestDBTaskCRUD(t *testing.T) {
	st := openTaskTestDB(t)
	ctx := t.Context()
	task := testTask()

	if err := CreateTask(ctx, st.RW(), task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	got, err := GetTask(ctx, st.RO(), task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.ID != task.ID || got.Definition != task.Definition || got.Specification != task.Specification ||
		got.State != task.State || got.Importance != task.Importance || got.Urgency != task.Urgency ||
		got.Complexity != task.Complexity || got.Effort != task.Effort || got.Risk != task.Risk ||
		got.Autonomy != task.Autonomy || got.CommitHash != task.CommitHash || got.Model != task.Model {
		t.Fatalf("GetTask = %+v, want %+v", got, task)
	}
	if got.FailureReason != "" {
		t.Fatalf("FailureReason = %q, want empty", got.FailureReason)
	}
	if len(got.Labels) != 2 || got.Labels[0] != "backend" || got.Labels[1] != "urgent" {
		t.Fatalf("labels = %v, want sorted labels", got.Labels)
	}

	task.State = TaskStateCompleted
	task.Labels = []string{"done"}
	task.CommitHash = "def456"
	if err := UpdateTask(ctx, st.RW(), task); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	got, err = GetTask(ctx, st.RO(), task.ID)
	if err != nil {
		t.Fatalf("GetTask after update: %v", err)
	}
	if got.State != task.State || got.CommitHash != task.CommitHash || len(got.Labels) != 1 || got.Labels[0] != "done" {
		t.Fatalf("updated task = %+v, want %+v", got, task)
	}

	task.State = TaskStateFailed
	task.FailureReason = "review rejected after fix attempt"
	if err := UpdateTask(ctx, st.RW(), task); err != nil {
		t.Fatalf("UpdateTask failed state: %v", err)
	}
	got, err = GetTask(ctx, st.RO(), task.ID)
	if err != nil {
		t.Fatalf("GetTask failed task: %v", err)
	}
	if got.State != TaskStateFailed || got.FailureReason != task.FailureReason {
		t.Fatalf("failed task = %+v, want state %q and reason %q", got, task.State, task.FailureReason)
	}

	if err := DeleteTask(ctx, st.RW(), task.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if _, err := GetTask(ctx, st.RO(), task.ID); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("GetTask deleted error = %v, want %v", err, ErrTaskNotFound)
	}
}

func TestDBReviewResultCRUD(t *testing.T) {
	st := openTaskTestDB(t)
	ctx := t.Context()
	task := testTask()
	if err := CreateTask(ctx, st.RW(), task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	first, err := InsertReviewResult(ctx, st.RW(), ReviewResult{
		TaskID:   task.ID,
		Approved: false,
		Findings: []ReviewFinding{{File: "main.go", Summary: "missing error check"}},
	})
	if err != nil {
		t.Fatalf("InsertReviewResult (1): %v", err)
	}
	if first.Attempt != 1 {
		t.Fatalf("first.Attempt = %d, want 1", first.Attempt)
	}

	second, err := InsertReviewResult(ctx, st.RW(), ReviewResult{
		TaskID:   task.ID,
		Approved: true,
	})
	if err != nil {
		t.Fatalf("InsertReviewResult (2): %v", err)
	}
	if second.Attempt != 2 {
		t.Fatalf("second.Attempt = %d, want 2", second.Attempt)
	}

	results, err := ReviewResultsForTask(ctx, st.RO(), task.ID)
	if err != nil {
		t.Fatalf("ReviewResultsForTask: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Approved || len(results[0].Findings) != 1 || results[0].Findings[0].File != "main.go" {
		t.Fatalf("results[0] = %+v, want unapproved with one finding", results[0])
	}
	if !results[1].Approved || len(results[1].Findings) != 0 {
		t.Fatalf("results[1] = %+v, want approved with no findings", results[1])
	}
}

func TestDBLinkSession(t *testing.T) {
	st := openTaskTestDB(t)
	ctx := t.Context()
	task := testTask()
	if err := CreateTask(ctx, st.RW(), task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	session1 := seedSession(t, st.RW())
	session2 := seedSession(t, st.RW())

	if err := LinkSession(ctx, st.RW(), task.ID, session1); err != nil {
		t.Fatalf("LinkSession (1): %v", err)
	}
	if err := LinkSession(ctx, st.RW(), task.ID, session2); err != nil {
		t.Fatalf("LinkSession (2): %v", err)
	}

	ids, err := SessionIDsForTask(ctx, st.RO(), task.ID)
	if err != nil {
		t.Fatalf("SessionIDsForTask: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("len(ids) = %d, want 2: %v", len(ids), ids)
	}
}

func TestDBListTasksFilterValues(t *testing.T) {
	st := openTaskTestDB(t)
	ctx := t.Context()

	first := testTask()
	second := testTask()
	second.ID = uuid.NewV7()
	second.State = TaskStateStarted
	second.Labels = []string{"frontend"}
	second.CommitHash = "def456"
	second.Importance = LevelLow
	third := testTask()
	third.ID = uuid.NewV7()
	third.State = TaskStateCompleted
	third.Labels = []string{"backend"}
	third.CommitHash = "ghi789"
	for _, task := range []Task{first, second, third} {
		if err := CreateTask(ctx, st.RW(), task); err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
	}

	tests := []struct {
		name   string
		filter TaskFilter
		want   []uuid.UUID
	}{
		{name: "one state", filter: TaskFilter{State: []TaskState{TaskStateStarted}}, want: []uuid.UUID{second.ID}},
		{
			name:   "many states",
			filter: TaskFilter{State: []TaskState{TaskStateCreated, TaskStateStarted}},
			want:   []uuid.UUID{first.ID, second.ID},
		},
		{name: "one level", filter: TaskFilter{Importance: []Level{LevelHigh}}, want: []uuid.UUID{first.ID, third.ID}},
		{name: "one label", filter: TaskFilter{Labels: []string{"frontend"}}, want: []uuid.UUID{second.ID}},
		{
			name:   "many commit hashes",
			filter: TaskFilter{CommitHashes: []string{"abc123", "ghi789"}},
			want:   []uuid.UUID{first.ID, third.ID},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListTasks(ctx, st.RO(), tt.filter)
			if err != nil {
				t.Fatalf("ListTasks: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(tasks) = %d, want %d: %+v", len(got), len(tt.want), got)
			}
			for i, wantID := range tt.want {
				if got[i].ID != wantID {
					t.Errorf("tasks[%d].ID = %s, want %s", i, got[i].ID, wantID)
				}
			}
		})
	}
}
