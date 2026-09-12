package escalated

import (
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return st
}

func TestEscalated(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(t.Context(), id, task.TaskDefinition{Description: "d"}, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := Escalated(t.Context(), st, Request{ID: id, Stage: "definition", Reason: "stuck"})
	if err != nil {
		t.Fatalf("Escalated() error = %v", err)
	}
	if got.State() != task.StateBlocked {
		t.Fatalf("state = %s, want %s", got.State(), task.StateBlocked)
	}
}

func TestEscalatedRejectsMissingStage(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := Escalated(t.Context(), st, Request{ID: id, Reason: "stuck"}); err == nil {
		t.Fatal("Escalated() error = nil, want an error for a missing stage")
	}
}
