package create

import (
	"path/filepath"
	"testing"

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

func TestCreate(t *testing.T) {
	st := openTestStore(t)
	got, err := Create(t.Context(), st, Request{Title: "t", Description: "do the work", Labels: map[string]string{"k": "v"}})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.Definition.Description != "do the work" || got.State() != task.StateSpecify {
		t.Fatalf("Create() = %+v", got)
	}
}

func TestCreateRejectsMissingDescription(t *testing.T) {
	st := openTestStore(t)
	if _, err := Create(t.Context(), st, Request{}); err == nil {
		t.Fatal("Create() error = nil, want an error for a missing description")
	}
}
