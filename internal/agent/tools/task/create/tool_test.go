package create

import (
	"database/sql"
	"path/filepath"
	"testing"
	"uuid"

	"github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/store"
	"github.com/mhmdkzr/loop/migrations"
	"github.com/mhmdkzr/loop/pkg/migrate"
	"github.com/mhmdkzr/loop/pkg/testenv"
)

func openTestStore(t *testing.T) *store.Store {
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

// seedModel inserts a model row under the "opencode" provider - the one
// sessions.ModelIDByName resolves against - so a task's Model can validate
// against something real.
func seedModel(t *testing.T, db *sql.DB, modelName string) {
	t.Helper()
	ctx := t.Context()
	providerID := uuid.NewV7()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO model_providers (provider_id, provider_name) VALUES (?, ?)`,
		providerID.String(), "opencode"); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO models (model_id, provider_id, model_name, context_window, has_vision) VALUES (?, ?, ?, 0, 0)`,
		uuid.NewV7().String(), providerID.String(), modelName); err != nil {
		t.Fatalf("insert model: %v", err)
	}
}

func TestInputValidate(t *testing.T) {
	valid := Input{
		Definition: "define work", Specification: "implement it",
		Model: "test-model", Importance: task.LevelHigh,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	for name, input := range map[string]Input{
		"missing definition":    {Specification: "implement it", Model: "test-model"},
		"missing specification": {Definition: "define work", Model: "test-model"},
		"missing model":         {Definition: "define work", Specification: "implement it"},
		"invalid level": {
			Definition: "define work", Specification: "implement it",
			Model: "test-model", Risk: task.Level(99),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := input.Validate(); err == nil {
				t.Fatal("Validate() returned nil")
			}
		})
	}
}

func TestExecuteCreatesTask(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	st := openTestStore(t)
	seedModel(t, st.RW(), "test-model")
	in := Input{
		Definition:    "define work",
		Specification: "implement it",
		Labels:        []string{"backend"},
		Importance:    task.LevelHigh,
		Model:         "test-model",
		CommitHash:    "abc123",
	}

	out, err := execute(t.Context(), st, in)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if out.ID == (uuid.UUID{}) {
		t.Fatal("execute returned zero ID")
	}
	if out.State != task.TaskStateCreated {
		t.Fatalf("state = %q, want %q", out.State, task.TaskStateCreated)
	}

	got, err := task.GetTask(t.Context(), st.RO(), out.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Definition != in.Definition || got.Specification != in.Specification || got.CommitHash != in.CommitHash {
		t.Fatalf("task = %+v, want fields from input", got)
	}
}
