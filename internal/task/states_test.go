package task

import "testing"

func TestParseTaskStateRejectsUnknownState(t *testing.T) {
	_, err := ParseTaskState("unknown")
	if err == nil {
		t.Fatal("ParseTaskState() error = nil, want an error")
	}
}
