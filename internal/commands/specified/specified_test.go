package specified

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

func TestSpecified(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(id, task.TaskDefinition{Description: "d"}, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := Specified(st, Request{ID: id, Plan: "do it"})
	if err != nil {
		t.Fatalf("Specified() error = %v", err)
	}
	if got.State() != task.StateImplement {
		t.Fatalf("state = %s, want %s", got.State(), task.StateImplement)
	}
}

func TestSpecifiedRejectsMissingPlan(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := st.Create(id, task.TaskDefinition{Description: "d"}, time.Now().UTC()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Specified(st, Request{ID: id}); err == nil {
		t.Fatal("Specified() error = nil, want an error for a missing plan")
	}
}
