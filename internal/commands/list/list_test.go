package list

import (
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
	"github.com/mhmdkzr/taskman/internal/utils"
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

func createTask(t *testing.T, st *store.Store, definition task.TaskDefinition, at time.Time) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	if _, err := st.Create(t.Context(), id, definition, at); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return id
}

func TestList(t *testing.T) {
	st := openTestStore(t)
	now := time.Now().UTC()
	createTask(t, st, task.TaskDefinition{Title: "t", Description: "d1"}, now)
	createTask(t, st, task.TaskDefinition{Title: "t", Description: "d2"}, now.Add(time.Second))

	tasks, total, err := List(t.Context(), st, Request{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 2 || total != 2 {
		t.Fatalf("List() = %d tasks, total %d, want 2 and 2", len(tasks), total)
	}
}

func TestListOnEmptyStoreReturnsEmpty(t *testing.T) {
	st := openTestStore(t)
	tasks, total, err := List(t.Context(), st, Request{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 0 || total != 0 {
		t.Fatalf("List() = %v (total %d), want empty", tasks, total)
	}
}

func TestListFiltersByState(t *testing.T) {
	st := openTestStore(t)
	now := time.Now().UTC()
	createTask(t, st, task.TaskDefinition{Title: "t", Description: "d"}, now)
	advanced := createTask(t, st, task.TaskDefinition{Title: "t", Description: "d"}, now.Add(time.Second))
	if _, err := st.Append(t.Context(), advanced, task.SpecificationSubmitted{
		Specification: task.Specification{Plan: "p"},
		At:            now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	tasks, total, err := List(t.Context(), st, Request{States: []task.TaskState{task.StateImplement}})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 1 || total != 1 || tasks[0].ID != advanced {
		t.Fatalf("List(implement) = %v (total %d), want only %s", tasks, total, advanced)
	}
}

func TestListFiltersByLabel(t *testing.T) {
	st := openTestStore(t)
	now := time.Now().UTC()
	createTask(t, st, task.TaskDefinition{Title: "t", Description: "d"}, now)
	labeled := createTask(t, st, task.TaskDefinition{
		Title:       "t",
		Description: "d",
		Labels:      map[string]string{"priority": "high"},
	}, now.Add(time.Second))

	tasks, total, err := List(t.Context(), st, Request{Labels: map[string]string{"priority": "high"}})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tasks) != 1 || total != 1 || tasks[0].ID != labeled {
		t.Fatalf("List(priority=high) = %v (total %d), want only %s", tasks, total, labeled)
	}
}

func TestListPaginates(t *testing.T) {
	st := openTestStore(t)
	now := time.Now().UTC()
	for i := range 3 {
		createTask(t, st, task.TaskDefinition{Title: "t", Description: "d"}, now.Add(time.Duration(i)*time.Second))
	}

	page, total, err := List(t.Context(), st, Request{Limit: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page) != 2 || total != 3 {
		t.Fatalf("List(limit 2) = %d tasks, total %d, want 2 and 3", len(page), total)
	}

	rest, total, err := List(t.Context(), st, Request{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(rest) != 1 || total != 3 {
		t.Fatalf("List(limit 2, offset 2) = %d tasks, total %d, want 1 and 3", len(rest), total)
	}
	if rest[0].ID == page[0].ID || rest[0].ID == page[1].ID {
		t.Fatalf("List() pages overlap: first %v, second %v", page, rest)
	}
}

func TestListRejectsUnknownState(t *testing.T) {
	st := openTestStore(t)
	if _, _, err := List(t.Context(), st, Request{States: []task.TaskState{"bogus"}}); err == nil {
		t.Fatal("List(bogus state) error = nil, want error")
	}
}

func TestMCPOutputSchemaIsObject(t *testing.T) {
	schema := utils.SchemaFor[Result]()
	if schema.Type != "object" {
		t.Fatalf("task_list output schema type = %q, want %q", schema.Type, "object")
	}
}
