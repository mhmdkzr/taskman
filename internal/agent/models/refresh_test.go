package models

import (
	"encoding/json"
	"testing"
)

func TestThinkingOptions(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{name: "generic", want: []string{"low", "medium", "high"}},
		{name: "deepseek-v4-flash", want: []string{"low", "medium", "high", "max"}},
		{name: "glm-5.2", want: []string{"high", "max"}},
		{name: "minimax-m3", want: []string{"disabled", "thinking"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := thinkingOptions(test.name)
			if len(got) != len(test.want) {
				t.Fatalf("option count = %d, want %d", len(got), len(test.want))
			}
			for _, name := range test.want {
				if _, ok := got[name]; !ok {
					t.Errorf("missing option %q", name)
				}
			}
		})
	}
}

func TestThinkingOptionJSON(t *testing.T) {
	for name, option := range thinkingOptions("deepseek-v4-flash") {
		var value map[string]string
		if err := json.Unmarshal(option, &value); err != nil {
			t.Fatalf("option %q is invalid JSON: %v", name, err)
		}
		if value["reasoning_effort"] == "" {
			t.Fatalf("option %q has no reasoning_effort", name)
		}
	}
}
