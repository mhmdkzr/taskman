package utils

import (
	"errors"
	"testing"

	"github.com/mhmdkzr/taskman/internal/task"
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

func TestParseCheckResult(t *testing.T) {
	got, err := ParseCheckResult("unit", "ok")
	if err != nil {
		t.Fatalf("ParseCheckResult: %v", err)
	}
	if got != task.CheckOK {
		t.Fatalf("got %q, want %q", got, task.CheckOK)
	}
	if got, err := ParseCheckResult("unit", ""); err != nil || got != "" {
		t.Fatalf("empty value: got (%q, %v), want (\"\", nil)", got, err)
	}
	if _, err := ParseCheckResult("unit", "maybe"); err == nil {
		t.Fatal("invalid value: want error")
	}
}

func TestFailMapsErrorsToExitCodes(t *testing.T) {
	if got := ExitCode(Fail(errors.New("boom"))); got != 1 {
		t.Errorf("Fail exit = %d, want 1", got)
	}
}
