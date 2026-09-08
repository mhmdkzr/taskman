package list

import (
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

func TestListEmpty(t *testing.T) {
	dir := t.TempDir()
	got, err := List(dir, Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty dir: got %v, want nil or empty slice", got)
	}
}

func TestListMultipleSorted(t *testing.T) {
	dir := t.TempDir()
	tasks := []task.Task{
		{
			ID:         "zebra",
			State:      task.StateCreated,
			Title:      "task z",
			Definition: "do z",
			Status:     task.Status{},
			Labels:     map[string]string{},
		},
		{
			ID:         "apple",
			State:      task.StateCreated,
			Title:      "task a",
			Definition: "do a",
			Status:     task.Status{},
			Labels:     map[string]string{},
		},
		{
			ID:         "middle",
			State:      task.StateCreated,
			Title:      "task m",
			Definition: "do m",
			Status:     task.Status{},
			Labels:     map[string]string{},
		},
	}
	for _, tk := range tasks {
		if err := task.WriteTaskFile(dir, tk); err != nil {
			t.Fatalf("write task file: %v", err)
		}
	}

	got, err := List(dir, Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d tasks, want 3", len(got))
	}
	if got[0].ID != "apple" || got[1].ID != "middle" || got[2].ID != "zebra" {
		t.Fatalf("tasks not sorted: got %v, %v, %v, want apple, middle, zebra", got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestListFilterByState(t *testing.T) {
	dir := t.TempDir()
	tasks := []task.Task{
		{
			ID:         "created_task",
			State:      task.StateCreated,
			Title:      "t1",
			Definition: "def",
			Status:     task.Status{},
			Labels:     map[string]string{},
		},
		{
			ID:         "started_task",
			State:      task.StateStarted,
			Title:      "t2",
			Definition: "def",
			Status:     task.Status{},
			Labels:     map[string]string{},
		},
		{
			ID:         "completed_task",
			State:      task.StateCompleted,
			Title:      "t3",
			Definition: "def",
			Status:     task.Status{},
			Labels:     map[string]string{},
		},
	}
	for _, tk := range tasks {
		if err := task.WriteTaskFile(dir, tk); err != nil {
			t.Fatalf("write task file: %v", err)
		}
	}

	filter := Filter{State: []task.State{task.StateStarted, task.StateCompleted}}
	got, err := List(dir, filter)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d tasks, want 2", len(got))
	}
	if got[0].ID != "completed_task" || got[1].ID != "started_task" {
		t.Fatalf("got wrong tasks: %v, %v", got[0].ID, got[1].ID)
	}
}

func TestListFilterByLabels(t *testing.T) {
	dir := t.TempDir()
	tasks := []task.Task{
		{
			ID:         "with_label",
			State:      task.StateCreated,
			Title:      "t1",
			Definition: "def",
			Status:     task.Status{},
			Labels:     map[string]string{"priority": "high"},
		},
		{
			ID:         "without_label",
			State:      task.StateCreated,
			Title:      "t2",
			Definition: "def",
			Status:     task.Status{},
			Labels:     map[string]string{},
		},
		{
			ID:         "different_label",
			State:      task.StateCreated,
			Title:      "t3",
			Definition: "def",
			Status:     task.Status{},
			Labels:     map[string]string{"priority": "low"},
		},
	}
	for _, tk := range tasks {
		if err := task.WriteTaskFile(dir, tk); err != nil {
			t.Fatalf("write task file: %v", err)
		}
	}

	filter := Filter{Labels: map[string]string{"priority": "high"}}
	got, err := List(dir, filter)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d tasks, want 1", len(got))
	}
	if got[0].ID != "with_label" {
		t.Fatalf("got wrong task: %v", got[0].ID)
	}
}

func TestListFilterByLabelsMissing(t *testing.T) {
	dir := t.TempDir()
	task1 := task.Task{
		ID:         "task1",
		State:      task.StateCreated,
		Title:      "t1",
		Definition: "def",
		Status:     task.Status{},
		Labels:     map[string]string{"priority": "high", "team": "backend"},
	}
	if err := task.WriteTaskFile(dir, task1); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	filter := Filter{Labels: map[string]string{"priority": "high", "team": "frontend"}}
	got, err := List(dir, filter)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d tasks, want 0 (label value mismatch)", len(got))
	}
}

func TestListFilterByLabelsMissingKey(t *testing.T) {
	dir := t.TempDir()
	task1 := task.Task{
		ID:         "task1",
		State:      task.StateCreated,
		Title:      "t1",
		Definition: "def",
		Status:     task.Status{},
		Labels:     map[string]string{"priority": "high"},
	}
	if err := task.WriteTaskFile(dir, task1); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	filter := Filter{Labels: map[string]string{"team": "backend"}}
	got, err := List(dir, filter)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d tasks, want 0 (label key missing)", len(got))
	}
}
