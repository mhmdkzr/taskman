package list

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
)

func TestMain(m *testing.M) {
	cli.OsExiter = func(int) {}
	os.Exit(m.Run())
}

func runCmd(t *testing.T, tasksDir string, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root := &cli.Command{
		Name: "taskman",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "tasks-dir", Value: tasksDir},
			&cli.BoolFlag{Name: "json"},
		},
		Commands: []*cli.Command{Command()},
	}
	root.Writer = &buf
	root.ErrWriter = &buf
	for _, sub := range root.Commands {
		sub.Writer = &buf
		sub.ErrWriter = &buf
	}
	full := append([]string{"taskman", "list"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommandEmptyDir(t *testing.T) {
	dir := t.TempDir()
	out, err := runCmd(t, dir)
	if err != nil {
		t.Fatalf("list: %v\noutput:\n%s", err, out)
	}
	if out != "no tasks\n" {
		t.Fatalf("empty dir output: got %q, want %q", out, "no tasks\n")
	}
}

func TestCommandListJSON(t *testing.T) {
	dir := t.TempDir()
	task1 := task.Task{
		ID:         "abc",
		State:      task.StateCreated,
		Title:      "task 1",
		Definition: "def",
		Status:     task.Status{},
		Labels:     map[string]string{"priority": "high"},
	}
	task2 := task.Task{
		ID:         "def",
		State:      task.StateStarted,
		Title:      "task 2",
		Definition: "def2",
		Status:     task.Status{},
		Labels:     map[string]string{},
	}
	if err := task.WriteTaskFile(dir, task1); err != nil {
		t.Fatalf("write task 1: %v", err)
	}
	if err := task.WriteTaskFile(dir, task2); err != nil {
		t.Fatalf("write task 2: %v", err)
	}

	out, err := runCmd(t, dir, "--json")
	if err != nil {
		t.Fatalf("list: %v\noutput:\n%s", err, out)
	}

	var result Result
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal json: %v\noutput:\n%s", err, out)
	}
	if len(result.Tasks) != 2 || result.Total != 2 {
		t.Fatalf("got %+v, want 2 tasks, total 2", result)
	}
	if result.Tasks[0].ID != "abc" || result.Tasks[1].ID != "def" {
		t.Fatalf("got wrong task ids: %v, %v", result.Tasks[0].ID, result.Tasks[1].ID)
	}
}

func TestCommandFilterByLabel(t *testing.T) {
	dir := t.TempDir()
	task1 := task.Task{
		ID:         "abc",
		State:      task.StateCreated,
		Title:      "task 1",
		Definition: "def",
		Status:     task.Status{},
		Labels:     map[string]string{"priority": "high"},
	}
	task2 := task.Task{
		ID:         "def",
		State:      task.StateStarted,
		Title:      "task 2",
		Definition: "def2",
		Status:     task.Status{},
		Labels:     map[string]string{"priority": "low"},
	}
	if err := task.WriteTaskFile(dir, task1); err != nil {
		t.Fatalf("write task 1: %v", err)
	}
	if err := task.WriteTaskFile(dir, task2); err != nil {
		t.Fatalf("write task 2: %v", err)
	}

	out, err := runCmd(t, dir, "--label", "priority=high", "--json")
	if err != nil {
		t.Fatalf("list: %v\noutput:\n%s", err, out)
	}

	var result Result
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal json: %v\noutput:\n%s", err, out)
	}
	if len(result.Tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(result.Tasks))
	}
	if result.Tasks[0].ID != "abc" {
		t.Fatalf("got wrong task id: %v", result.Tasks[0].ID)
	}
}

func TestCommandPagination(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{"a", "b", "c"} {
		tk := task.Task{ID: id, State: task.StateCreated, Title: id, Definition: "def", Status: task.Status{}}
		if err := task.WriteTaskFile(dir, tk); err != nil {
			t.Fatalf("write task %s: %v", id, err)
		}
	}

	out, err := runCmd(t, dir, "--limit", "2", "--json")
	if err != nil {
		t.Fatalf("list: %v\noutput:\n%s", err, out)
	}
	var result Result
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("unmarshal json: %v\noutput:\n%s", err, out)
	}
	if len(result.Tasks) != 2 || result.Total != 3 || result.Limit != 2 || result.Offset != 0 {
		t.Fatalf("got %+v, want 2 tasks, total 3, limit 2, offset 0", result)
	}

	page2, err := runCmd(t, dir, "--limit", "2", "--offset", "2", "--json")
	if err != nil {
		t.Fatalf("list: %v\noutput:\n%s", err, page2)
	}
	var result2 Result
	if err := json.Unmarshal([]byte(page2), &result2); err != nil {
		t.Fatalf("unmarshal json: %v\noutput:\n%s", err, page2)
	}
	if len(result2.Tasks) != 1 || result2.Tasks[0].ID != "c" {
		t.Fatalf("got %+v, want just task c", result2)
	}

	humanOut, err := runCmd(t, dir, "--limit", "2")
	if err != nil {
		t.Fatalf("list: %v\noutput:\n%s", err, humanOut)
	}
	if !strings.Contains(humanOut, "1 more") {
		t.Fatalf("human output = %q, want a hint about the remaining task", humanOut)
	}
}
