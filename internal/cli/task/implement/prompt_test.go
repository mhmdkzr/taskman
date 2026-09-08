package implement

import (
	"strings"
	"testing"
)

func TestPromptRender(t *testing.T) {
	got := Prompt{
		Definition:    "def",
		Specification: "spec",
		DoneWhen:      "criteria",
		References:    []string{"a.go:1"},
	}.Render()
	for _, want := range []string{"def", "spec", "criteria", "a.go:1"} {
		if !strings.Contains(got, want) {
			t.Errorf("Render() missing %q, got:\n%s", want, got)
		}
	}
}
