package update

import (
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateStarted,
		Title:      "original title",
		Definition: "def",
		Labels: map[string]string{
			"team": "backend",
		},
		References: []string{"ref1", "ref2"},
		Status: task.Status{
			Definition: task.StageStatus{State: task.StageDone},
		},
	}
	if err := task.WriteTaskFile(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestUpdateTitle(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	newTitle := "updated title"
	req := Request{
		Title: &newTitle,
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Title != "updated title" {
		t.Fatalf("title = %q, want %q", got.Title, "updated title")
	}
	// Verify persisted
	read, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if read.Title != "updated title" {
		t.Fatalf("persisted title = %q, want %q", read.Title, "updated title")
	}
}

func TestUpdateSetLabels(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		SetLabels: map[string]string{
			"priority": "high",
		},
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Labels["priority"] != "high" {
		t.Fatalf("priority label = %q, want %q", got.Labels["priority"], "high")
	}
	// Existing label should remain
	if got.Labels["team"] != "backend" {
		t.Fatalf("team label = %q, want %q", got.Labels["team"], "backend")
	}
}

func TestUpdateUnsetLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		UnsetLabels: []string{"team"},
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok := got.Labels["team"]; ok {
		t.Fatal("team label should be unset")
	}
}

func TestUpdateSetAndUnsetLabels(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		SetLabels: map[string]string{
			"priority": "medium",
		},
		UnsetLabels: []string{"team"},
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, ok := got.Labels["team"]; ok {
		t.Fatal("team label should be unset")
	}
	if got.Labels["priority"] != "medium" {
		t.Fatalf("priority label = %q, want %q", got.Labels["priority"], "medium")
	}
}

func TestUpdateInvalidLabelValue(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		SetLabels: map[string]string{
			"priority": "urgent", // invalid, must be low/medium/high
		},
	}
	_, err := Update(dir, "abc", req)
	if err == nil {
		t.Fatal("update with invalid priority: want error, got nil")
	}
}

func TestUpdateComplexityLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		SetLabels: map[string]string{
			"complexity": "low",
		},
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Labels["complexity"] != "low" {
		t.Fatalf("complexity label = %q, want %q", got.Labels["complexity"], "low")
	}
}

func TestUpdateAutonomyLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		SetLabels: map[string]string{
			"autonomy": "high",
		},
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Labels["autonomy"] != "high" {
		t.Fatalf("autonomy label = %q, want %q", got.Labels["autonomy"], "high")
	}
}

func TestUpdateUserDefinedLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		SetLabels: map[string]string{
			"custom_tag": "any_value",
		},
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Labels["custom_tag"] != "any_value" {
		t.Fatalf("custom_tag label = %q, want %q", got.Labels["custom_tag"], "any_value")
	}
}

func TestUpdateSetReferences(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		References: []string{"new_ref1", "new_ref2", "new_ref3"},
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(got.References) != 3 {
		t.Fatalf("references length = %d, want 3", len(got.References))
	}
	if got.References[0] != "new_ref1" {
		t.Fatalf("references[0] = %q, want %q", got.References[0], "new_ref1")
	}
}

func TestUpdateClearReferences(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		ClearReferences: true,
	}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(got.References) != 0 {
		t.Fatalf("references length = %d, want 0", len(got.References))
	}
}

func TestUpdateNoChange(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{}
	got, err := Update(dir, "abc", req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Title != "original title" {
		t.Fatalf("title should not change, got %q", got.Title)
	}
	if got.Labels["team"] != "backend" {
		t.Fatalf("labels should not change, got %q", got.Labels["team"])
	}
}
