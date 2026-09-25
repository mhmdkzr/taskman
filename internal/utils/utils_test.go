package utils

import (
	"context"
	"testing"

	"github.com/urfave/cli/v3"
)

// runWithFlags parses args against a command declaring name/value flags (all
// optional) and returns whatever the given check returns, to exercise
// RequireFlags against real cmd.IsSet state rather than a hand-built struct.
func runWithFlags(t *testing.T, flagNames []string, args []string, check func(*cli.Command) error) error {
	t.Helper()
	flags := make([]cli.Flag, len(flagNames))
	for i, name := range flagNames {
		flags[i] = &cli.StringFlag{Name: name}
	}
	var result error
	cmd := &cli.Command{
		Name:  "test",
		Flags: flags,
		Action: func(_ context.Context, cmd *cli.Command) error {
			result = check(cmd)
			return nil
		},
	}
	if err := cmd.Run(t.Context(), append([]string{"test"}, args...)); err != nil {
		t.Fatalf("cmd.Run() error = %v", err)
	}
	return result
}

func TestRequireFlagsAllSet(t *testing.T) {
	err := runWithFlags(t, []string{"a", "b"}, []string{"--a", "1", "--b", "2"}, func(cmd *cli.Command) error {
		return RequireFlags(cmd, "a", "b")
	})
	if err != nil {
		t.Fatalf("RequireFlags() error = %v, want nil", err)
	}
}

func TestRequireFlagsOneMissing(t *testing.T) {
	err := runWithFlags(t, []string{"a", "b"}, []string{"--a", "1"}, func(cmd *cli.Command) error {
		return RequireFlags(cmd, "a", "b")
	})
	if err == nil {
		t.Fatal("RequireFlags() error = nil, want an error for missing --b")
	}
	if exitErr, ok := err.(cli.ExitCoder); !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("RequireFlags() = %v, want a cli.ExitCoder with code 2", err)
	}
}

func TestRequireFlagsMultipleMissing(t *testing.T) {
	err := runWithFlags(t, []string{"a", "b", "c"}, nil, func(cmd *cli.Command) error {
		return RequireFlags(cmd, "a", "b", "c")
	})
	if err == nil {
		t.Fatal("RequireFlags() error = nil, want an error for missing --a, --b, --c")
	}
	if exitErr, ok := err.(cli.ExitCoder); !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("RequireFlags() = %v, want a cli.ExitCoder with code 2", err)
	}
}

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
