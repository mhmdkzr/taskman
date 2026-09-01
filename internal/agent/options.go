// Package agent provides a library API for running an LLM agent with tools,
// in both streaming and non-streaming modes. It wraps github.com/zendev-sh/goai
// behind agent-owned types so callers never touch provider-specific APIs.
package agent

import (
	"fmt"

	"github.com/mhmdkzr/taskman/internal/codebase"
	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/tools"
	"github.com/mhmdkzr/taskman/internal/tools/schedule"
	"github.com/mhmdkzr/taskman/internal/tools/spawn"
	tasktools "github.com/mhmdkzr/taskman/internal/tools/task"
	"github.com/mhmdkzr/taskman/internal/tools/telegram"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
)

// RunRequest, RunSubject and MeaningfulOverride are defined in the schedule
// package, which owns the agent.run wire contract; they are re-exported here so
// consumers of this package keep a single import.
const RunSubject = schedule.RunSubject

type RunRequest = schedule.RunRequest

// MeaningfulOverride trims v and returns it unchanged, or "" when it is empty
// or the sentinel "default". Scheduling agents often emit "default" in optional
// fields to mean "inherit", so consumers treat it like an omitted value.
func MeaningfulOverride(v string) string { return schedule.MeaningfulOverride(v) }

// ReasoningEffort controls how much reasoning the model performs.
type ReasoningEffort string

const (
	ReasoningEffortNone   ReasoningEffort = "none"
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortHigh   ReasoningEffort = "high"
	ReasoningEffortMax    ReasoningEffort = "max"
)

const (
	DefaultModel           = "deepseek-v4-flash"
	DefaultReasoningEffort = ReasoningEffortMedium
	DefaultMaxSteps        = 100
	// DefaultMaxSubagentDepth is the default nesting limit for spawn_subagent.
	// The top-level agent is depth 0; its children are depth 1, and so on.
	DefaultMaxSubagentDepth = 4
)

// Options configures an agent. Zero values fall back to safe defaults. The
// event bus is not part of Options: the publisher and scheduler are passed
// separately to the functions that need them.
type Options struct {
	// Config carries the provider credentials and DB path.
	Config config.AgentConfig

	// model overrides the provider model. Unexported: only tests set it.
	model provider.LanguageModel

	// Model is the language model ID to use. Empty uses DefaultModel.
	Model string

	// Tools are the tools the agent can call. Nil builds the default set (see
	// DefaultTools) when a session is created.
	Tools []goai.Tool

	// SystemPrompt is the system instruction. Empty sends no system message.
	SystemPrompt string

	// ReasoningEffort controls reasoning depth. Empty uses ReasoningEffortLow.
	ReasoningEffort ReasoningEffort

	// MaxSteps bounds the automatic tool loop. Non-positive uses DefaultMaxSteps.
	MaxSteps int

	// MaxSubagentDepth bounds how deep spawn_subagent can nest. Non-positive
	// uses DefaultMaxSubagentDepth.
	MaxSubagentDepth int

	// Codebase is the git repository directory the read/edit/glob/grep tools
	// and go build/test/vet operate on. Empty defaults to ".". A task
	// pipeline run sets this to that task's own worktree, so its tools can
	// only touch that checkout; the default suits the general-purpose
	// (non-task) agent runtime, which operates on the process's own repo.
	Codebase string
}

// DefaultOptions returns the fully-normalized default options: default model,
// reasoning effort, max steps, sub-agent depth and DB path. Provider
// credentials must be supplied separately (they are read from the environment
// by the application config and passed via OptionsFromConfig).
func DefaultOptions() (Options, error) {
	return normalize(Options{})
}

// OptionsFromConfig returns the fully-normalized options carrying the given
// env-loaded config.
func OptionsFromConfig(cfg config.AgentConfig) (Options, error) {
	return normalize(Options{Config: cfg})
}

// SetModelForTesting overrides the provider model sessions use. It exists so
// tests outside the agent package (e.g. the server integration tests) can run
// against a fake model; production entry points never set it.
func (o *Options) SetModelForTesting(m provider.LanguageModel) { o.model = m }

// DefaultTools builds the default tool set for a Store and options: the
// codebase tools (read, edit, glob, grep) scoped to o.Codebase, the task
// backlog tools (create, search, get, edit) backed by st, the telegram tools
// when credentials are configured, and the spawn_subagent / subagent_result
// tools backed by a fresh sub-agent Runner. It returns the tools and the
// Runner, which the caller can Wait on so in-flight sub-agents finish before
// the process exits.
func DefaultTools(st *store.Store, pub publisher.Publisher, o Options) ([]goai.Tool, *spawn.Runner, error) {
	codebaseDir := o.Codebase
	if codebaseDir == "" {
		codebaseDir = "."
	}
	repo, err := codebase.Open(codebaseDir)
	if err != nil {
		return nil, nil, fmt.Errorf("open codebase %q: %w", codebaseDir, err)
	}
	tg, err := telegram.NewClientFromConfig(o.Config.Telegram)
	if err != nil {
		return nil, nil, err
	}
	base := tools.Tools(repo, tg)
	base = append(base, tasktools.Tools(st.RW())...)
	maxDepth := o.MaxSubagentDepth
	if maxDepth <= 0 {
		maxDepth = DefaultMaxSubagentDepth
	}
	runner := newSubagentRunner(st, pub, o, base, maxDepth)
	full := make([]goai.Tool, 0, len(base)+2)
	full = append(full, base...)
	full = append(full, runner.SpawnTool(1), runner.ResultTool())
	return full, runner, nil
}

func normalize(o Options) (Options, error) {
	if o.Config.DBPath == "" {
		o.Config.DBPath = config.DefaultDBPath
	}
	dbPath, err := config.ExpandHome(o.Config.DBPath)
	if err != nil {
		return o, err
	}
	o.Config.DBPath = dbPath
	if o.Model == "" {
		o.Model = DefaultModel
	}
	if o.ReasoningEffort == "" {
		o.ReasoningEffort = DefaultReasoningEffort
	}
	if o.MaxSteps <= 0 {
		o.MaxSteps = DefaultMaxSteps
	}
	if o.Codebase == "" {
		o.Codebase = "."
	}
	return o, nil
}
