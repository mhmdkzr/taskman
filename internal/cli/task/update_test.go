package task

import (
	"encoding/json"
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

func getJSON(t *testing.T, gitDir, id string) task.Task {
	t.Helper()
	out, err := runCmd(t, gitDir, Get(), id, "--json")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	var tk task.Task
	if err := json.Unmarshal([]byte(out), &tk); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	return tk
}

func TestUpdateLabels(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)

	if _, err := runCmd(t, dir, Update(), id, "--label", "priority=high", "--label", "complexity=low"); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := getJSON(t, dir, id)
	if got.Labels["priority"] != "high" || got.Labels["complexity"] != "low" {
		t.Fatalf("labels = %+v", got.Labels)
	}

	if _, err := runCmd(t, dir, Update(), id, "--unset-label", "complexity"); err != nil {
		t.Fatalf("update unset: %v", err)
	}
	got = getJSON(t, dir, id)
	if _, ok := got.Labels["complexity"]; ok {
		t.Fatalf("labels = %+v, want complexity removed", got.Labels)
	}
	if got.Labels["priority"] != "high" {
		t.Fatalf("labels = %+v, want priority kept", got.Labels)
	}
}

func TestUpdateReferences(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)

	if _, err := runCmd(t, dir, Update(), id, "--reference", "a.go:1", "--reference", "b.go:2"); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := getJSON(t, dir, id)
	if len(got.References) != 2 {
		t.Fatalf("references = %v, want 2", got.References)
	}

	if _, err := runCmd(t, dir, Update(), id, "--clear-references"); err != nil {
		t.Fatalf("update clear: %v", err)
	}
	got = getJSON(t, dir, id)
	if len(got.References) != 0 {
		t.Fatalf("references = %v, want none", got.References)
	}
}

func TestUpdateInvalidLabel(t *testing.T) {
	dir := newTestRepo(t)
	id := createTask(t, dir)
	if _, err := runCmd(t, dir, Update(), id, "--label", "priority=urgent"); err == nil {
		t.Fatal("update with invalid priority: want error, got nil")
	}
}
