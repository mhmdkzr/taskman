// Package utils holds the plumbing shared by more than one taskman CLI
// command slice: parsing the --id and repeated flag values, the shared
// review-gate flag set, MCP JSON Schema inference, and error wrapping. It
// has no dependency on internal/commands or any slice under it, so any of
// them can import it without a cycle.
package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"uuid"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/urfave/cli/v3"

	"github.com/mhmdkzr/taskman/internal/task"
)

// IDFrom parses the --id flag as a task id.
func IDFrom(cmd *cli.Command) (uuid.UUID, error) {
	raw := cmd.String("id")
	if raw == "" {
		return uuid.Nil(), cli.Exit("--id is required", 2)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil(), cli.Exit(fmt.Sprintf("--id: %v", err), 2)
	}
	return id, nil
}

// RequireFlagsIf calls RequireFlags only when cond is true - for flags that
// are required together only conditional on another flag (e.g. --worktree/
// --branch are required together only when --use-worktree is set).
func RequireFlagsIf(cmd *cli.Command, cond bool, names ...string) error {
	if !cond {
		return nil
	}
	return RequireFlags(cmd, names...)
}

// RequireFlags returns a cli.Exit error (exit code 2) naming every flag in
// names that cmd did not receive, or nil if all were set. Every command
// calls this itself instead of declaring its flags Required: true: urfave/
// cli's own required-flag check fails with an unexported error type
// (errRequiredFlags) that implements neither cli.ExitCoder nor any exported
// interface, so nothing downstream can distinguish it from a domain error
// via errors.Is/As - only a string match on its message ever could. Calling
// RequireFlags explicitly keeps every malformed-input error a cli.Exit,
// consistent with IDFrom above, with no message-matching required anywhere.
func RequireFlags(cmd *cli.Command, names ...string) error {
	var missing []string
	for _, name := range names {
		if !cmd.IsSet(name) {
			missing = append(missing, name)
		}
	}
	switch len(missing) {
	case 0:
		return nil
	case 1:
		return cli.Exit(fmt.Sprintf("--%s is required", missing[0]), 2)
	default:
		flags := make([]string, len(missing))
		for i, name := range missing {
			flags[i] = "--" + name
		}
		return cli.Exit(fmt.Sprintf("%s are required", strings.Join(flags, ", ")), 2)
	}
}

// SplitKV parses "key=value" pairs (e.g. --label priority=high). value may
// itself contain "=" - only the first separator counts.
func SplitKV(pairs []string) (map[string]string, error) {
	out := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, fmt.Errorf("malformed key=value pair: %q", pair)
		}
		out[key] = value
	}
	return out, nil
}

// ParseFindings parses repeated --finding location=detail pairs.
func ParseFindings(pairs []string) ([]task.Finding, error) {
	kv, err := SplitKV(pairs)
	if err != nil {
		return nil, err
	}
	findings := make([]task.Finding, 0, len(kv))
	for location, detail := range kv {
		findings = append(findings, task.Finding{Location: location, Detail: detail})
	}
	return findings, nil
}

// ReviewFlags are shared by every command that reports a review gate's
// configuration (specified, for both the specification-review gate and,
// prefixed, the implementation-review gate): whether an agent/human review
// is required, and each one's auto-fix policy. prefix distinguishes multiple
// review gates declared on the same command (e.g. "" for the spec-review
// gate and "impl-" for the implementation-review gate, both set at
// `specified` time) - empty prefix reproduces the original flag names.
func ReviewFlags(prefix string) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: prefix + "agent-review", Usage: "require an automated review"},
		&cli.BoolFlag{Name: prefix + "agent-review-use-subagent", Usage: "run the automated review in a subagent"},
		&cli.BoolFlag{Name: prefix + "agent-review-auto-fix", Usage: "automatically fix automated review findings"},
		&cli.IntFlag{Name: prefix + "agent-review-auto-fix-max-rounds", Usage: "max automated-review auto-fix rounds"},
		&cli.BoolFlag{
			Name: prefix + "agent-review-auto-fix-use-subagent", Usage: "run automated-review auto-fix in a subagent",
		},
		&cli.BoolFlag{Name: prefix + "human-review", Usage: "require a human review"},
		&cli.BoolFlag{Name: prefix + "human-review-auto-fix", Usage: "automatically fix human review findings"},
		&cli.IntFlag{Name: prefix + "human-review-auto-fix-max-rounds", Usage: "max human-review auto-fix rounds"},
		&cli.BoolFlag{
			Name: prefix + "human-review-auto-fix-use-subagent", Usage: "run human-review auto-fix in a subagent",
		},
	}
}

// ReviewConfigurationFrom builds a task.ReviewConfiguration from the flags a
// matching ReviewFlags(prefix) call declared.
func ReviewConfigurationFrom(cmd *cli.Command, prefix string) task.ReviewConfiguration {
	return task.ReviewConfiguration{
		Agent: task.AgentReviewConfiguration{
			Required:    cmd.Bool(prefix + "agent-review"),
			UseSubagent: cmd.Bool(prefix + "agent-review-use-subagent"),
			AutoFix: task.AutoFix{
				Enabled:     cmd.Bool(prefix + "agent-review-auto-fix"),
				MaxRounds:   cmd.Int(prefix + "agent-review-auto-fix-max-rounds"),
				UseSubagent: cmd.Bool(prefix + "agent-review-auto-fix-use-subagent"),
			},
		},
		Human: task.HumanReviewConfiguration{
			Required: cmd.Bool(prefix + "human-review"),
			AutoFix: task.AutoFix{
				Enabled:     cmd.Bool(prefix + "human-review-auto-fix"),
				MaxRounds:   cmd.Int(prefix + "human-review-auto-fix-max-rounds"),
				UseSubagent: cmd.Bool(prefix + "human-review-auto-fix-use-subagent"),
			},
		},
	}
}

// SchemaFor infers a JSON Schema for T with taskman's uuid type registered
// explicitly. The MCP SDK's reflection infers uuid.UUID ([16]byte) as an
// array of integers, but JSON encodes it as a string (uuid implements
// encoding.TextMarshaler), so without this a tool's schema would reject its
// own wire format on both input and output.
func SchemaFor[T any]() *jsonschema.Schema {
	s, err := jsonschema.ForType(reflect.TypeFor[T](), &jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[uuid.UUID](): {Type: "string", Format: "uuid"},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("infer json schema for %T: %v", *new(T), err))
	}
	return s
}

// Fail wraps a domain error into a cli.ExitCoder. Flag-parsing/usage errors
// (e.g. from IDFrom or a slice's local flag parsing) already carry exit code
// 2 and are returned unchanged; every other domain error reaching here is a
// runtime failure, exit code 1.
func Fail(err error) error {
	if exitErr, ok := errors.AsType[cli.ExitCoder](err); ok {
		return exitErr
	}
	return cli.Exit(err, 1)
}
