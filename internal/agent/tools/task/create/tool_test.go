package create

import (
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

func TestInputValidate(t *testing.T) {
	valid := Input{Definition: "define work", Specification: "implement it", Importance: task.LevelHigh}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	for name, input := range map[string]Input{
		"missing definition":    {Specification: "implement it"},
		"missing specification": {Definition: "define work"},
		"invalid level":         {Definition: "define work", Specification: "implement it", Risk: task.Level(99)},
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
