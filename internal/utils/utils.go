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
// configuration (specified, implemented): whether an agent/human review is
// required, and each one's auto-fix policy.
func ReviewFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "agent-review", Usage: "require an automated review"},
		&cli.BoolFlag{Name: "agent-review-use-subagent", Usage: "run the automated review in a subagent"},
		&cli.BoolFlag{Name: "agent-review-auto-fix", Usage: "automatically fix automated review findings"},
		&cli.IntFlag{Name: "agent-review-auto-fix-max-rounds", Usage: "max automated-review auto-fix rounds"},
		&cli.BoolFlag{Name: "agent-review-auto-fix-use-subagent", Usage: "run automated-review auto-fix in a subagent"},
		&cli.BoolFlag{Name: "human-review", Usage: "require a human review"},
		&cli.BoolFlag{Name: "human-review-auto-fix", Usage: "automatically fix human review findings"},
		&cli.IntFlag{Name: "human-review-auto-fix-max-rounds", Usage: "max human-review auto-fix rounds"},
		&cli.BoolFlag{Name: "human-review-auto-fix-use-subagent", Usage: "run human-review auto-fix in a subagent"},
	}
}

// ReviewConfigurationFrom builds a task.ReviewConfiguration from the flags
// ReviewFlags declares.
func ReviewConfigurationFrom(cmd *cli.Command) task.ReviewConfiguration {
	return task.ReviewConfiguration{
		Agent: task.AgentReviewConfiguration{
			Required:    cmd.Bool("agent-review"),
			UseSubagent: cmd.Bool("agent-review-use-subagent"),
			AutoFix: task.AutoFix{
				Enabled:     cmd.Bool("agent-review-auto-fix"),
				MaxRounds:   cmd.Int("agent-review-auto-fix-max-rounds"),
				UseSubagent: cmd.Bool("agent-review-auto-fix-use-subagent"),
			},
		},
		Human: task.HumanReviewConfiguration{
			Required: cmd.Bool("human-review"),
			AutoFix: task.AutoFix{
				Enabled:     cmd.Bool("human-review-auto-fix"),
				MaxRounds:   cmd.Int("human-review-auto-fix-max-rounds"),
				UseSubagent: cmd.Bool("human-review-auto-fix-use-subagent"),
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
