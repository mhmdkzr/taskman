package task

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/mhmdkzr/loop/internal/task"
)

func commitTaskFiles(t *testing.T, gitDir string) {
	t.Helper()
	for _, args := range [][]string{
		{"add", ".tasks"},
		{"commit", "-q", "-m", "chore: track task"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = gitDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func TestListEmpty(t *testing.T) {
	dir := newTestRepo(t)
	out, err := runCmd(t, dir, List())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if out != "no tasks\n" {
		t.Errorf("list output = %q, want %q", out, "no tasks\n")
	}
}

func TestListAndFilterByLabel(t *testing.T) {
	dir := newTestRepo(t)
	createTask(t, dir)
	commitTaskFiles(t, dir)
	if _, err := runCmd(
		t,
		dir,
		Create(),
		"--definition",
		"two",
		"--title",
		"Two",
		"--label",
		"priority=high",
	); err != nil {
		t.Fatalf("create: %v", err)
	}

	out, err := runCmd(t, dir, List(), "--json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var tasks []task.Task
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("unmarshal: %v\noutput:\n%s", err, out)
	}
	if len(tasks) != 2 {
		t.Fatalf("list len = %d, want 2", len(tasks))
	}

	filtered, err := runCmd(t, dir, List(), "--label", "priority=high", "--json")
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	var filteredTasks []task.Task
	if err := json.Unmarshal([]byte(filtered), &filteredTasks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(filteredTasks) != 1 || filteredTasks[0].Title != "Two" {
		t.Fatalf("filtered = %+v, want just Two", filteredTasks)
	}
}
