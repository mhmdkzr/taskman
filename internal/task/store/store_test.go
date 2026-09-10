package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
)

func TestWriteReadAndUpdate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	original := task.Task{ID: "abc", State: task.StateSpecify, Definition: "definition"}
	if err := Write(dir, original); err != nil {
		t.Fatalf("Write: %v", err)
	}
	read, err := Read(dir, "abc")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if read.State != task.StateSpecify {
		t.Fatalf("state = %q", read.State)
	}
	data, err := os.ReadFile(filepath.Join(dir, "abc.yaml"))
	if err != nil {
		t.Fatalf("read task file: %v", err)
	}
	if strings.Contains(string(data), "schema_version") {
		t.Fatal("task file contains schema_version")
	}
	updated, err := Update(dir, "abc", func(current task.Task) (task.Task, error) {
		return task.Apply(current, task.SpecificationSubmitted{Specification: "spec", DoneWhen: "done"})
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.State != task.StateImplement {
		t.Fatalf("state = %q", updated.State)
	}
}

func TestReadRejectsUnknownState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "abc.yaml"),
		[]byte("task:\n  id: abc\n  state: created\n  definition: work\n"),
		0o600,
	); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := Read(dir, "abc")
	if !errors.Is(err, task.ErrUnknownState) {
		t.Fatalf("error = %v, want ErrUnknownState", err)
	}
}
