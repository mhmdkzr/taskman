package list

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/loop/internal/task"
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

	var tasks []task.Task
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("unmarshal json: %v\noutput:\n%s", err, out)
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2", len(tasks))
	}
	if tasks[0].ID != "abc" || tasks[1].ID != "def" {
		t.Fatalf("got wrong task ids: %v, %v", tasks[0].ID, tasks[1].ID)
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

	var tasks []task.Task
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("unmarshal json: %v\noutput:\n%s", err, out)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}
	if tasks[0].ID != "abc" {
		t.Fatalf("got wrong task id: %v", tasks[0].ID)
	}
}
