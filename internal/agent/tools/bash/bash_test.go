package bash

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	result, err := execute(context.Background(), Input{Command: "printf 'hello'; printf 'warning' >&2"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.StdOut != "hello" {
		t.Errorf("stdout = %q, want %q", result.StdOut, "hello")
	}
	if result.StdErr != "warning" {
		t.Errorf("stderr = %q, want %q", result.StdErr, "warning")
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestExecuteFailure(t *testing.T) {
	result, err := execute(context.Background(), Input{Command: "printf 'failure' >&2; exit 7"})
	if err == nil {
		t.Fatal("execute succeeded, want error")
	}
	if !strings.Contains(err.Error(), "bash:") {
		t.Errorf("error = %q, want bash context", err)
	}
	if result.ExitCode != 7 {
		t.Errorf("exit code = %d, want 7", result.ExitCode)
	}
	if result.StdErr != "failure" {
		t.Errorf("stderr = %q, want %q", result.StdErr, "failure")
	}
}

func TestInputValidate(t *testing.T) {
	if err := (Input{}).Validate(); !errors.Is(err, ErrEmptyCommand) {
		t.Errorf("error = %v, want %v", err, ErrEmptyCommand)
	}
	if err := (Input{Command: " \\t"}).Validate(); err != nil {
		t.Errorf("whitespace command rejected: %v", err)
	}
}
