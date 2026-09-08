package task

import (
	"strings"
	"testing"
	"uuid"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Fix Doc Drift":         "fix-doc-drift",
		"  leading/trailing  ":  "leading-trailing",
		"Balance's API (v2)!!!": "balance-s-api-v2",
		"":                      "task",
		"!!!":                   "task",
		"already-hyphenated":    "already-hyphenated",
		"MiXeD CaSe 123":        "mixed-case-123",
	}
	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Errorf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID("Fix Doc Drift")
	parts := strings.SplitN(id, "_", 2)
	if len(parts) != 2 {
		t.Fatalf("id = %q, want <uuid>_<slug>", id)
	}
	if parts[1] != "fix-doc-drift" {
		t.Errorf("slug part = %q, want fix-doc-drift", parts[1])
	}
	if _, err := uuid.Parse(parts[0]); err != nil {
		t.Errorf("uuid part = %q, not a valid uuid: %v", parts[0], err)
	}

	id2 := GenerateID("Fix Doc Drift")
	if id == id2 {
		t.Error("GenerateID returned the same id twice - uuid v7 should never collide in a test run")
	}
}
