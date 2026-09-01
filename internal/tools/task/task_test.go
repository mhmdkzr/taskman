package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/task"
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

func exec(t *testing.T, tool goai.Tool, input any) (string, error) {
	t.Helper()
	b, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	return tool.Execute(context.Background(), b)
}

func validCreateInput() createInput {
	return createInput{
		Title:         "Fix nil pointer",
		TaskType:      "bug",
		Urgency:       "low",
		Importance:    "medium",
		Risk:          "low",
		What:          "Guard against a nil client before calling Send.",
		Why:           "Send panics when the client is nil.",
		How:           "Add a nil check at the top of Send.",
		Packages:      []string{"internal/foo"},
		CompletedWhen: []string{"tests pass"},
	}
}

func TestToolsCount(t *testing.T) {
	if got := len(Tools(newTestDB(t))); got != 4 {
		t.Errorf("tools = %d, want 4 (create, search, get, edit)", got)
	}
}

func TestCreateToolRejectsInvalidInput(t *testing.T) {
	db := newTestDB(t)
	create := CreateTool(db)

	cases := []struct {
		name   string
		mutate func(in *createInput)
	}{
		{"bad task_type", func(in *createInput) { in.TaskType = "chore" }},
		{"bad urgency", func(in *createInput) { in.Urgency = "urgent" }},
		{"bad risk", func(in *createInput) { in.Risk = "extreme" }},
		{"missing title", func(in *createInput) { in.Title = "" }},
		{"nil packages", func(in *createInput) { in.Packages = nil }},
		{"nil completed_when", func(in *createInput) { in.CompletedWhen = nil }},
		{"bad variant", func(in *createInput) { in.Variant = "ultra" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := validCreateInput()
			c.mutate(&in)
			if _, err := exec(t, create, in); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

// TestCreateSearchGetEdit walks a task through the tool set end to end:
// create it, find it via search, read it back in full, then edit its spec.
func TestCreateSearchGetEdit(t *testing.T) {
	db := newTestDB(t)
	create, search, get, editT := CreateTool(db), SearchTool(db), GetTool(db), EditTool(db)

	out, err := exec(t, create, validCreateInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out, "Fix nil pointer") {
		t.Errorf("create output = %q, want it to mention the title", out)
	}

	tasks, err := task.SearchTasks(context.Background(), db, task.TaskFilter{})
	if err != nil || len(tasks) != 1 {
		t.Fatalf("expected exactly one task after create, got %v (err=%v)", tasks, err)
	}
	id := tasks[0].ID.String()

	searchOut, err := exec(t, search, searchInput{TaskType: "bug"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(searchOut, id) {
		t.Errorf("search output = %q, want it to contain id %s", searchOut, id)
	}

	getOut, err := exec(t, get, getInput{ID: id})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(getOut, `"status":"created"`) {
		t.Errorf("get output = %q, want status created", getOut)
	}

	editIn := editInput{
		ID:            id,
		Title:         "Fix nil pointer in Send",
		TaskType:      "bug",
		Urgency:       "low",
		Importance:    "medium",
		Risk:          "high",
		What:          "Guard against a nil client before calling Send.",
		Why:           "Send panics when the client is nil.",
		How:           "Add a nil check at the top of Send.",
		Packages:      []string{"internal/foo", "internal/bar"},
		CompletedWhen: []string{"tests pass"},
	}
	if _, err := exec(t, editT, editIn); err != nil {
		t.Fatalf("edit: %v", err)
	}

	got, err := task.GetTask(context.Background(), db, tasks[0].ID)
	if err != nil {
		t.Fatalf("GetTask after edit: %v", err)
	}
	if got.Title != "Fix nil pointer in Send" || got.Risk != "high" || len(got.Packages) != 2 {
		t.Errorf("after edit = %+v", got)
	}
}

func TestGetToolRejectsInvalidID(t *testing.T) {
	db := newTestDB(t)
	if _, err := exec(t, GetTool(db), getInput{ID: "not-a-uuid"}); err == nil {
		t.Error("want error for invalid id")
	}
}

func TestSearchToolRejectsInvalidFilter(t *testing.T) {
	db := newTestDB(t)
	if _, err := exec(t, SearchTool(db), searchInput{Status: "done"}); err == nil {
		t.Error("want error for invalid status filter")
	}
}
