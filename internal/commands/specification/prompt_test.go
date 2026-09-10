package specification

import (
	"strings"
	"testing"
)

func TestPromptRender(t *testing.T) {
	got := Prompt{Definition: "def", References: []string{"a.go:1", "b.go:2"}}.Render()
	for _, want := range []string{"def", "a.go:1", "b.go:2"} {
		if !strings.Contains(got, want) {
			t.Errorf("Render() missing %q, got:\n%s", want, got)
		}
	}
}
