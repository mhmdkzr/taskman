package update

import (
	"bytes"
	"context"
	"os"
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
	full := append([]string{"taskman", "update"}, args...)
	err := root.Run(context.Background(), full)
	return buf.String(), err
}

func TestCommandUpdateTitle(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc", "--title", "new title")
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.Title != "new title" {
		t.Fatalf("title = %q, want %q", got.Title, "new title")
	}
}

func TestCommandUpdateLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc", "--label", "priority=high")
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.Labels["priority"] != "high" {
		t.Fatalf("priority label = %q, want %q", got.Labels["priority"], "high")
	}
}

func TestCommandUpdateMultipleLabels(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc",
		"--label", "priority=medium",
		"--label", "complexity=high",
	)
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.Labels["priority"] != "medium" {
		t.Fatalf("priority label = %q, want %q", got.Labels["priority"], "medium")
	}
	if got.Labels["complexity"] != "high" {
		t.Fatalf("complexity label = %q, want %q", got.Labels["complexity"], "high")
	}
}

func TestCommandUnsetLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc", "--unset-label", "team")
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if _, ok := got.Labels["team"]; ok {
		t.Fatal("team label should be unset")
	}
}

func TestCommandSetAndUnsetLabels(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc",
		"--label", "priority=low",
		"--unset-label", "team",
	)
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if _, ok := got.Labels["team"]; ok {
		t.Fatal("team label should be unset")
	}
	if got.Labels["priority"] != "low" {
		t.Fatalf("priority label = %q, want %q", got.Labels["priority"], "low")
	}
}

func TestCommandInvalidLabelValue(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	_, err := runCmd(t, dir, "abc", "--label", "priority=urgent")
	if err == nil {
		t.Fatal("update with invalid priority: want error, got nil")
	}
}

func TestCommandSetReferences(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc",
		"--reference", "new_ref1",
		"--reference", "new_ref2",
	)
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if len(got.References) != 2 {
		t.Fatalf("references length = %d, want 2", len(got.References))
	}
	if got.References[0] != "new_ref1" {
		t.Fatalf("references[0] = %q, want %q", got.References[0], "new_ref1")
	}
}

func TestCommandClearReferences(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc", "--clear-references")
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if len(got.References) != 0 {
		t.Fatalf("references length = %d, want 0", len(got.References))
	}
}

func TestCommandTitleAndLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc",
		"--title", "combined update",
		"--label", "autonomy=medium",
	)
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.Title != "combined update" {
		t.Fatalf("title = %q, want %q", got.Title, "combined update")
	}
	if got.Labels["autonomy"] != "medium" {
		t.Fatalf("autonomy label = %q, want %q", got.Labels["autonomy"], "medium")
	}
}

func TestCommandSetTrunk(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc", "--trunk")
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if !got.Git.Trunk {
		t.Fatal("Git.Trunk = false, want true")
	}
}

func TestCommandUnsetTrunk(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	if _, err := runCmd(t, dir, "abc", "--trunk"); err != nil {
		t.Fatalf("set trunk: %v", err)
	}
	out, err := runCmd(t, dir, "abc", "--trunk=false")
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.Git.Trunk {
		t.Fatal("Git.Trunk = true, want false")
	}
}

func TestCommandSetAutoApprove(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	out, err := runCmd(t, dir, "abc", "--auto-approve")
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if !got.AutoApprove {
		t.Fatal("AutoApprove = false, want true")
	}
}

func TestCommandUnsetAutoApprove(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	if _, err := runCmd(t, dir, "abc", "--auto-approve"); err != nil {
		t.Fatalf("set auto-approve: %v", err)
	}
	out, err := runCmd(t, dir, "abc", "--auto-approve=false")
	if err != nil {
		t.Fatalf("update: %v\noutput:\n%s", err, out)
	}
	got, err := task.ReadTask(dir, "abc")
	if err != nil {
		t.Fatalf("read task: %v", err)
	}
	if got.AutoApprove {
		t.Fatal("AutoApprove = true, want false")
	}
}

func TestCommandMissingID(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, dir)
	if err == nil {
		t.Fatal("update without an id: want error, got nil")
	}
}

func TestCommandTaskNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, dir, "nonexistent")
	if err == nil {
		t.Fatal("update nonexistent task: want error, got nil")
	}
}

func TestCommandMalformedLabel(t *testing.T) {
	dir := newTestTaskDir(t, "abc")
	_, err := runCmd(t, dir, "abc", "--label", "no_equals_sign")
	if err == nil {
		t.Fatal("update with malformed label: want error, got nil")
	}
}
