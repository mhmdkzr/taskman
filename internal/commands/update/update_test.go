package update

import (
	"errors"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
	"github.com/mhmdkzr/taskman/internal/task/store"
)

//nolint:unparam // id is always "abc" in this file, but keeping it explicit reads better than a magic string inside the helper
func newTestTaskDir(t *testing.T, id string) string {
	t.Helper()
	dir := t.TempDir()
	tk := task.Task{
		ID:         id,
		State:      task.StateImplement,
		Title:      "original title",
		Definition: "def",
		Labels: map[string]string{
			"team": "backend",
		},
		References: []string{"ref1", "ref2"},
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	return dir
}

func TestUpdateTitle(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	newTitle := "updated title"
	req := Request{
		ID:    "abc",
		Title: &newTitle,
	}
	got, err := Update(dir, req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Title != "updated title" {
		t.Fatalf("title = %q, want %q", got.Title, "updated title")
	}
	// Verify persisted
	read, err := store.Read(dir, "abc")
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
		ID: "abc",
		SetLabels: map[string]string{
			"priority": "high",
		},
	}
	got, err := Update(dir, req)
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
		ID:          "abc",
		UnsetLabels: []string{"team"},
	}
	got, err := Update(dir, req)
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
		ID: "abc",
		SetLabels: map[string]string{
			"priority": "medium",
		},
		UnsetLabels: []string{"team"},
	}
	got, err := Update(dir, req)
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
		ID: "abc",
		SetLabels: map[string]string{
			"priority": "urgent", // invalid, must be low/medium/high
		},
	}
	_, err := Update(dir, req)
	if err == nil {
		t.Fatal("update with invalid priority: want error, got nil")
	}
}

func TestUpdateComplexityLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{
		ID: "abc",
		SetLabels: map[string]string{
			"complexity": "low",
		},
	}
	got, err := Update(dir, req)
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
		ID: "abc",
		SetLabels: map[string]string{
			"autonomy": "high",
		},
	}
	got, err := Update(dir, req)
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
		ID: "abc",
		SetLabels: map[string]string{
			"custom_tag": "any_value",
		},
	}
	got, err := Update(dir, req)
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
		ID:         "abc",
		References: []string{"new_ref1", "new_ref2", "new_ref3"},
	}
	got, err := Update(dir, req)
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
		ID:              "abc",
		ClearReferences: true,
	}
	got, err := Update(dir, req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(got.References) != 0 {
		t.Fatalf("references length = %d, want 0", len(got.References))
	}
}

func TestUpdateSetTrunk(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	trunk := true
	req := Request{ID: "abc", Trunk: &trunk}
	got, err := Update(dir, req)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !got.Git.Trunk {
		t.Fatal("Git.Trunk = false, want true")
	}
}

func TestUpdateUnsetTrunk(t *testing.T) {
	dir := t.TempDir()
	tk := task.Task{
		ID:         "abc",
		State:      task.StateImplement,
		Title:      "original title",
		Definition: "def",
		Git:        task.Git{Trunk: true},
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	trunk := false
	got, err := Update(dir, Request{ID: "abc", Trunk: &trunk})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Git.Trunk {
		t.Fatal("Git.Trunk = true, want false")
	}
}

func TestUpdateSetAutoApprove(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	autoApprove := true
	got, err := Update(dir, Request{ID: "abc", AutoApprove: &autoApprove})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !got.AutoApprove {
		t.Fatal("AutoApprove = false, want true")
	}
}

func TestUpdateUnsetAutoApprove(t *testing.T) {
	dir := t.TempDir()
	tk := task.Task{
		ID:          "abc",
		State:       task.StateImplement,
		Title:       "original title",
		Definition:  "def",
		AutoApprove: true,
	}
	if err := store.Write(dir, tk); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	autoApprove := false
	got, err := Update(dir, Request{ID: "abc", AutoApprove: &autoApprove})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.AutoApprove {
		t.Fatal("AutoApprove = true, want false")
	}
}

func TestUpdateAutoApproveRejectedWhenMoot(t *testing.T) {
	for _, state := range []task.State{
		task.StateHumanReview,
		task.StateMerge,
		task.StateBlocked,
		task.StateCompleted,
		task.StateAbandoned,
	} {
		dir := t.TempDir()
		tk := task.Task{
			ID:         "abc",
			State:      state,
			Title:      "original title",
			Definition: "def",
		}
		if state == task.StateBlocked {
			tk.Blocked = &task.Blocked{ResumeState: task.StateHumanReview, Stage: "review", Reason: "stuck"}
		}
		if err := store.Write(dir, tk); err != nil {
			t.Fatalf("write task file: %v", err)
		}
		autoApprove := true
		_, err := Update(dir, Request{ID: "abc", AutoApprove: &autoApprove})
		if !errors.Is(err, task.ErrAutoApproveTooLate) {
			t.Errorf("state %s: Update err = %v, want ErrAutoApproveTooLate", state, err)
		}
		read, readErr := store.Read(dir, "abc")
		if readErr != nil {
			t.Fatalf("read task: %v", readErr)
		}
		if read.AutoApprove {
			t.Errorf("state %s: AutoApprove persisted as true, want unchanged (false)", state)
		}
	}
}

func TestUpdateNoChange(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	req := Request{ID: "abc"}
	got, err := Update(dir, req)
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
