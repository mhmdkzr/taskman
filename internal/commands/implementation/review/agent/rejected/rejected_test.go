package rejected

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

func TestRejected(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	now := time.Now().UTC()
	if _, err := st.Create(id, task.TaskDefinition{Description: "d"}, now); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := st.Append(id, task.SpecificationSubmitted{Specification: task.Specification{Plan: "p"}, At: now}); err != nil {
		t.Fatalf("Append(SpecificationSubmitted) error = %v", err)
	}
	if _, err := st.Append(id, task.ImplementationCompleted{
		Implementation: task.Implementation{Review: task.ReviewConfiguration{
			Agent: task.AgentReviewConfiguration{Required: true},
		}},
		At: now,
	}); err != nil {
		t.Fatalf("Append(ImplementationCompleted) error = %v", err)
	}

	got, err := Rejected(st, Request{ID: id, Findings: []task.Finding{{Location: "x", Detail: "y"}}})
	if err != nil {
		t.Fatalf("Rejected() error = %v", err)
	}
	if got.State() != task.StateFixAutomatedReviewFindings {
		t.Fatalf("state = %s, want %s", got.State(), task.StateFixAutomatedReviewFindings)
	}
}

func TestRejectedRejectsNoFindings(t *testing.T) {
	st := openTestStore(t)
	id := uuid.NewV7()
	if _, err := Rejected(st, Request{ID: id}); err == nil {
		t.Fatal("Rejected() error = nil, want an error for no findings")
	}
}
