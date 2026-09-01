package agent

import (
	"context"

	"github.com/mhmdkzr/taskman/internal/config"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/tools/spawn"
	"github.com/zendev-sh/goai"
)

// newSubagentRunner builds the process-wide sub-agent Runner for a tool set.
// base is the non-spawn tool set (bash, files, telegram); the runner's
// spawn tools append themselves and the result tool to every sub-agent's set,
// so nesting and inspection keep working. runSubagent is the executor that
// actually runs a sub-agent: a fresh Session with an isolated transcript.
func newSubagentRunner(st *store.Store, pub publisher.Publisher, o Options, base []goai.Tool, maxDepth int) *spawn.Runner {
	cfg := o.Config
	return spawn.NewRunner(st, spawn.ExecOptions{
		Model:           o.Model,
		ReasoningEffort: string(o.ReasoningEffort),
		MaxSteps:        o.MaxSteps,
		SystemPrompt:    o.SystemPrompt,
	}, pub, base, func(ctx context.Context, opts spawn.ExecOptions) (*spawn.ExecResult, error) {
		return runSubagent(ctx, st, cfg, pub, opts)
	}, maxDepth)
}

// runSubagent executes a sub-agent: a fresh session with its own transcript,
// running the given prompt with the given tool set. On success it returns the
// run in the store form the spawn Runner persists.
func runSubagent(ctx context.Context, st *store.Store, cfg config.AgentConfig, pub publisher.Publisher, opts spawn.ExecOptions) (*spawn.ExecResult, error) {
	o := Options{
		Config:          cfg,
		Model:           opts.Model,
		ReasoningEffort: ReasoningEffort(opts.ReasoningEffort),
		MaxSteps:        opts.MaxSteps,
		SystemPrompt:    opts.SystemPrompt,
		Tools:           opts.Tools,
	}
	s, err := NewSession(o, pub, opts.SessionID)
	if err != nil {
		return nil, err
	}
	res, err := s.Run(ctx, opts.Prompt)
	if err != nil {
		return nil, err
	}
	return &spawn.ExecResult{
		Text:     res.Text,
		Usage:    res.Usage,
		Messages: messagesToStore(res.Steps, res.Messages),
	}, nil
}
