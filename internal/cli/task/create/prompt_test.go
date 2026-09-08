package create

import (
	"strings"
	"testing"
)

func TestSummaryRender(t *testing.T) {
	got := Summary{
		TaskID:   "abc-123",
		Title:    "Test Task",
		Worktree: "/path/to/worktree",
		Branch:   "task/abc-123",
	}.Render()
	for _, want := range []string{"abc-123", "Test Task", "/path/to/worktree", "task/abc-123"} {
		if !strings.Contains(got, want) {
			t.Errorf("Render() missing %q, got:\n%s", want, got)
		}
	}
}
