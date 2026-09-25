package add

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
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "tasks.db"))
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

func createTestTask(t *testing.T, st *store.Store, labels map[string]string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	if _, err := st.Create(
		t.Context(),
		id,
		task.TaskDefinition{Title: "t", Description: "d", Labels: labels},
		time.Now().UTC(),
	); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return id
}

func TestAddSetsNewLabel(t *testing.T) {
	st := openTestStore(t)
	id := createTestTask(t, st, nil)

	got, err := Add(t.Context(), st, Request{ID: id, Labels: map[string]string{"priority": "high"}})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got.Definition.Labels["priority"] != "high" {
		t.Fatalf("labels = %v, want priority=high", got.Definition.Labels)
	}
	if got.State() != task.StateSpecify {
		t.Fatalf("state = %s, want unchanged %s", got.State(), task.StateSpecify)
	}
}

func TestAddOverwritesExistingLabel(t *testing.T) {
	st := openTestStore(t)
	id := createTestTask(t, st, map[string]string{"priority": "low"})

	got, err := Add(t.Context(), st, Request{ID: id, Labels: map[string]string{"priority": "high"}})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if got.Definition.Labels["priority"] != "high" {
		t.Fatalf("labels = %v, want priority=high", got.Definition.Labels)
	}
}

func TestAddRejectsEmptyLabels(t *testing.T) {
	st := openTestStore(t)
	id := createTestTask(t, st, nil)

	if _, err := Add(t.Context(), st, Request{ID: id}); err == nil {
		t.Fatal("Add() error = nil, want an error for no labels")
	}
}
