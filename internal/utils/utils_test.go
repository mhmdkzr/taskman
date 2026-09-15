package utils

import (
	"testing"
)

func TestSplitKV(t *testing.T) {
	got, err := SplitKV([]string{"a=1", "b=two=parts"})
	if err != nil {
		t.Fatalf("SplitKV: %v", err)
	}
	if got["a"] != "1" || got["b"] != "two=parts" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseFindings(t *testing.T) {
	got, err := ParseFindings([]string{"main.go=bad"})
	if err != nil {
		t.Fatalf("ParseFindings: %v", err)
	}
	if len(got) != 1 || got[0].Location != "main.go" || got[0].Detail != "bad" {
		t.Fatalf("got %#v", got)
	}
}
