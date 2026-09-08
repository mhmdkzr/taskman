package skill

import (
	"strings"
	"testing"
)

func TestContent(t *testing.T) {
	got := Content()
	if got == "" {
		t.Fatal("Content() is empty")
	}
	if !strings.Contains(got, "name: taskman") {
		t.Fatalf("Content() = %q, want it to contain the SKILL.md frontmatter", got)
	}
}
