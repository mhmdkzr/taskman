// Package spawn provides the spawn_subagent and subagent_result tools: an
// agent can delegate work to a sub-agent that runs outside its own context,
// with access to the full tool set (including the spawn tools, so sub-agents
// can nest). The spawner receives only the sub-agent's session id; the
// sub-agent's transcript lives in its own session in the shared store and is
// read back through subagent_result.
//
// A Runner is the process-wide registry: it is built once when the parent
// agent's tool set is assembled, and every spawned sub-agent's tool set
// includes a fresh spawn tool bound to the same Runner at a higher depth.
// Depth is bounded so runaway recursion fails fast with a tool error.
//
// Persistence reuses the session schema. Spawning creates the sub-agent's
// session row eagerly (so its id exists and status is derivable); the run is
// appended as a normal turn on completion; and failures are persisted as a
// turn whose single step carries finish_reason "error" (real model runs never
// set that value, so subagent_result can tell done from failed).
package spawn

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mhmdkzr/taskman/internal/events"
	"github.com/mhmdkzr/taskman/internal/publisher"
	"github.com/mhmdkzr/taskman/internal/store"
	"github.com/mhmdkzr/taskman/internal/usage"
	"github.com/zendev-sh/goai"
)

const (
	// SpawnToolName is the registered name of the spawn tool.
	SpawnToolName = "spawn_subagent"

	// ResultToolName is the registered name of the result tool.
	ResultToolName = "subagent_result"

	// errFinishReason marks a persisted error turn; real model runs never use it.
	errFinishReason = "error"

	// defaultCellLen bounds a single output cell when formatting results, so a
	// sub-agent's answer cannot flood the spawner's context window.
	defaultCellLen = 2000

	// maxWaitSeconds bounds wait_seconds in the result tool.
	maxWaitSeconds = 300

	// publishTimeout bounds each event publish so a stalled bus cannot hang a
	// tool call.
	publishTimeout = 5 * time.Second
)

// Executor runs a sub-agent to completion and returns what the Runner needs to
// persist it. It is supplied by the runtime (the agent package) so the spawn
// package never depends on agent internals.
type Executor func(ctx context.Context, opts ExecOptions) (*ExecResult, error)

// ExecOptions is the resolved configuration a sub-agent runs with. The Runner
// fills it from the spawn tool's input and the parent's defaults.
type ExecOptions struct {
	// SessionID is the pre-generated session the sub-agent runs in; its events
	// and persisted turn carry it.
	SessionID       string
	Model           string
	ReasoningEffort string
	MaxSteps        int
	SystemPrompt    string
	// Tools is the sub-agent's full tool set, including a spawn tool one level
	// deeper and the result tool, so nesting and inspection keep working.
	Tools  []goai.Tool
	Prompt string
}

// ExecResult is everything needed to persist a completed sub-agent run as one
// transaction (store.AppendRun). Messages are already converted to store form.
type ExecResult struct {
	Text     string
	Usage    usage.TokenUsage
	Messages []store.Message
}

// Runner is the process-wide sub-agent registry. It is safe for concurrent use.
type Runner struct {
	st        *store.Store
	base      ExecOptions
	baseTools []goai.Tool
	exec      Executor
	maxDepth  int
	pub       publisher.Publisher

	mu       sync.Mutex
	inflight map[string]chan struct{}
}

// NewRunner builds a Runner. base carries the parent agent's defaults that
// sub-agents inherit (model, effort, max steps, system prompt); pub publishes
// the sub-agent lifecycle events; baseTools is the non-spawn tool set shared
// by every sub-agent; exec runs a sub-agent; maxDepth bounds nesting (the
// top-level agent is depth 0, so its children are depth 1).
func NewRunner(st *store.Store, base ExecOptions, pub publisher.Publisher, baseTools []goai.Tool, exec Executor, maxDepth int) *Runner {
	if maxDepth < 1 {
		maxDepth = 1
	}
	return &Runner{
		st:        st,
		base:      base,
		pub:       pub,
		baseTools: baseTools,
		exec:      exec,
		maxDepth:  maxDepth,
		inflight:  make(map[string]chan struct{}),
	}
}

// SpawnTool returns a spawn_subagent tool whose children run at atDepth. The
// top-level agent holds SpawnTool(1); a sub-agent at depth d holds
// SpawnTool(d+1). Each call returns a fresh tool so depth is captured per
// level without mutable state.
func (r *Runner) SpawnTool(atDepth int) goai.Tool {
	return goai.NewTool(SpawnToolName,
		"Run a sub-agent in an isolated context. The sub-agent has the full tool set, including this tool, so it can nest. "+
			"This tool returns the sub-agent's session id immediately and the sub-agent runs asynchronously outside the caller's context. "+
			"Read its outcome with the subagent_result tool (pass the session_id). "+
			"Set wait=true to block until the sub-agent finishes and return its answer inline. "+
			"Give the sub-agent a self-contained prompt: it has no access to this conversation's history.",
		func(ctx context.Context, in spawnInput) (string, error) {
			return r.spawn(ctx, atDepth, in)
		})
}

// ResultTool returns a subagent_result tool bound to this Runner. It is
// stateless and may be shared across all depths.
func (r *Runner) ResultTool() goai.Tool {
	return goai.NewTool(ResultToolName,
		"Read the outcome of a sub-agent spawned with spawn_subagent. "+
			"Pass the sub-agent's session_id. Returns a running status while the sub-agent works, and its final answer (or an error) when done. "+
			"Set wait_seconds to block up to that long for completion instead of polling. "+
			"Set include_steps=true to include the sub-agent's tool calls.",
		func(ctx context.Context, in resultInput) (string, error) {
			return r.result(ctx, in)
		})
}

// Wait blocks until every in-flight sub-agent has finished (success, error or
// panic) and returns nil, or ctx.Err() if the context is cancelled first. It
// re-scans so sub-agents spawned by still-running sub-agents are awaited too.
func (r *Runner) Wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		snapshot := make([]chan struct{}, 0, len(r.inflight))
		for _, done := range r.inflight {
			snapshot = append(snapshot, done)
		}
		r.mu.Unlock()
		if len(snapshot) == 0 {
			return nil
		}
		for _, done := range snapshot {
			select {
			case <-done:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// spawnInput is the spawn_subagent tool's schema.
type spawnInput struct {
	Prompt          string `json:"prompt" jsonschema:"description=The task for the sub-agent. Must be self-contained: the sub-agent has no access to this conversation's history."`
	SystemPrompt    string `json:"system_prompt,omitempty" jsonschema:"description=Override the sub-agent's system prompt (defaults to this agent's)."`
	Model           string `json:"model,omitempty" jsonschema:"description=Override the sub-agent's model (defaults to this agent's)."`
	ReasoningEffort string `json:"reasoning_effort,omitempty" jsonschema:"description=Override reasoning effort: none, low, medium, high or max (defaults to this agent's)."`
	MaxSteps        *int   `json:"max_steps,omitempty" jsonschema:"description=Override the maximum tool-loop steps (defaults to this agent's)."`
	Wait            *bool  `json:"wait,omitempty" jsonschema:"description=When true, block until the sub-agent finishes and return its answer inline. When false (default), return the session id immediately."`
}

// spawn validates the request, resolves the sub-agent's options, creates its
// session eagerly, starts it in a background goroutine, and returns its id.
func (r *Runner) spawn(ctx context.Context, atDepth int, in spawnInput) (string, error) {
	if atDepth > r.maxDepth {
		return "", fmt.Errorf("%s: maximum sub-agent depth (%d) reached", SpawnToolName, r.maxDepth)
	}

	prompt := strings.TrimSpace(in.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("%s: prompt is required", SpawnToolName)
	}

	model := strings.TrimSpace(in.Model)
	if model == "" {
		model = r.base.Model
	}
	effort := strings.TrimSpace(in.ReasoningEffort)
	if effort == "" {
		effort = r.base.ReasoningEffort
	}
	maxSteps := r.base.MaxSteps
	if in.MaxSteps != nil && *in.MaxSteps > 0 {
		maxSteps = *in.MaxSteps
	}
	sysPrompt := in.SystemPrompt
	if strings.TrimSpace(sysPrompt) == "" {
		sysPrompt = r.base.SystemPrompt
	}

	childTools := r.childTools(atDepth)

	sessionID, err := store.NewSessionID()
	if err != nil {
		return "", fmt.Errorf("%s: generate session id: %w", SpawnToolName, err)
	}
	if err := store.CreateSession(ctx, r.st.RW(), sessionID, model, effort, maxSteps, sysPrompt, toolsToStore(childTools)); err != nil {
		return "", fmt.Errorf("%s: create session: %w", SpawnToolName, err)
	}
	r.publish(ctx, events.SubagentSpawned{
		SessionID: sessionID,
		Depth:     atDepth,
		Prompt:    prompt,
		Model:     model,
		Wait:      in.Wait != nil && *in.Wait,
	})

	opts := ExecOptions{
		SessionID:       sessionID,
		Model:           model,
		ReasoningEffort: effort,
		MaxSteps:        maxSteps,
		SystemPrompt:    sysPrompt,
		Tools:           childTools,
		Prompt:          prompt,
	}

	// The child runs detached from the parent's generation context so it
	// survives the parent's turn ending; the Runner's Wait is the lifecycle
	// join point.
	done := r.track(sessionID)
	go r.runChild(context.WithoutCancel(ctx), sessionID, opts, done)

	if in.Wait == nil || !*in.Wait {
		return fmt.Sprintf("sub-agent spawned: session %s (running)\n"+
			"read its result with subagent_result (session_id=%s).",
			sessionID, sessionID), nil
	}

	select {
	case <-done:
	case <-ctx.Done():
		return "", fmt.Errorf("%s: %w", SpawnToolName, ctx.Err())
	}
	return r.resultText(ctx, sessionID)
}

// childTools is the tool set a sub-agent at atDepth runs with: the shared base
// tools plus a spawn tool one level deeper and the result tool.
func (r *Runner) childTools(atDepth int) []goai.Tool {
	out := make([]goai.Tool, 0, len(r.baseTools)+2)
	out = append(out, r.baseTools...)
	out = append(out, r.SpawnTool(atDepth+1))
	out = append(out, r.ResultTool())
	return out
}

// runChild executes one sub-agent and persists its outcome as a turn of its
// session. Success is appended as a normal run; a returned error or panic is
// persisted as an error turn so the spawner can distinguish failure from a
// still-running sub-agent.
func (r *Runner) runChild(ctx context.Context, sessionID string, opts ExecOptions, done chan struct{}) {
	defer r.untrack(sessionID)
	defer func() {
		if p := recover(); p != nil {
			r.persistError(ctx, sessionID, opts.Prompt, fmt.Errorf("%s: panic: %v", SpawnToolName, p))
		}
	}()

	res, err := r.exec(ctx, opts)
	if err != nil {
		r.persistError(ctx, sessionID, opts.Prompt, err)
		return
	}
	run := store.Run{
		Prompt:   opts.Prompt,
		Messages: res.Messages,
	}
	if err := store.AppendRun(ctx, r.st.RW(), sessionID, run); err != nil {
		r.persistError(ctx, sessionID, opts.Prompt, err)
		return
	}
	r.publish(ctx, events.SubagentFinished{
		SessionID: sessionID,
		Text:      res.Text,
		Usage:     usageToEvent(res.Usage),
		Steps:     countMessages(res.Messages),
	})
}

// persistError records a failed sub-agent as a turn whose single message has
// finish_reason "error" and the failure text in it.
func (r *Runner) persistError(ctx context.Context, sessionID, prompt string, err error) {
	run := store.Run{
		Prompt: prompt,
		Messages: []store.Message{{
			Role:         "assistant",
			Number:       1,
			FinishReason: errFinishReason,
			Parts: []store.Part{{
				Type: "text",
				Text: fmt.Sprintf("sub-agent failed: %v", err),
			}},
		}},
	}
	_ = store.AppendRun(ctx, r.st.RW(), sessionID, run)
	r.publish(ctx, events.SubagentFailed{
		SessionID: sessionID,
		Prompt:    prompt,
		Error:     err.Error(),
	})
}

// publish sends an event on the bus with a bounded wait so a stalled bus
// cannot hang a tool call. Publish errors are best-effort and ignored.
func (r *Runner) publish(ctx context.Context, e publisher.Event[any]) {
	pubCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	_ = r.pub.Publish(pubCtx, e)
}

// usageToEvent converts a store usage to its event form.
func usageToEvent(u usage.TokenUsage) events.Usage {
	return events.Usage{
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		TotalTokens:      u.TotalTokens,
		ReasoningTokens:  u.ReasoningTokens,
		CacheReadTokens:  u.CacheReadTokens,
		CacheWriteTokens: u.CacheWriteTokens,
	}
}

// resultInput is the subagent_result tool's schema.
type resultInput struct {
	SessionID    string `json:"session_id" jsonschema:"description=The session id returned by spawn_subagent."`
	WaitSeconds  *int   `json:"wait_seconds,omitempty" jsonschema:"description=Maximum seconds to wait for the sub-agent to finish (default 0 = report the current status immediately)."`
	IncludeSteps *bool  `json:"include_steps,omitempty" jsonschema:"description=When true, include the sub-agent's steps and tool calls in the output."`
}

// result reports the status of a sub-agent and, once finished, its outcome.
func (r *Runner) result(ctx context.Context, in resultInput) (string, error) {
	sessionID := strings.TrimSpace(in.SessionID)
	if sessionID == "" {
		return "", fmt.Errorf("%s: session_id is required", ResultToolName)
	}

	wait := 0
	if in.WaitSeconds != nil {
		wait = *in.WaitSeconds
		if wait < 0 || wait > maxWaitSeconds {
			return "", fmt.Errorf("%s: wait_seconds must be between 0 and %d", ResultToolName, maxWaitSeconds)
		}
	}
	includeSteps := in.IncludeSteps != nil && *in.IncludeSteps
	deadline := time.Now().Add(time.Duration(wait) * time.Second)

	for {
		tr, err := store.GetSessionTranscript(ctx, r.st.RO(), sessionID)
		if err != nil {
			return "", fmt.Errorf("%s: %w", ResultToolName, err)
		}
		if len(tr.Turns) == 0 {
			if wait == 0 || time.Now().After(deadline) {
				return fmt.Sprintf("%s: sub-agent %s is still running; "+
					"re-check with this tool or set wait_seconds to block for completion", ResultToolName, sessionID), nil
			}
			if !sleepCtx(ctx, 250*time.Millisecond) {
				return "", fmt.Errorf("%s: %w", ResultToolName, ctx.Err())
			}
			continue
		}
		return formatTranscript(sessionID, tr, includeSteps), nil
	}
}

// resultText renders a finished sub-agent's outcome (used by spawn's wait
// mode, which has already confirmed the run is persisted).
func (r *Runner) resultText(ctx context.Context, sessionID string) (string, error) {
	tr, err := store.GetSessionTranscript(ctx, r.st.RO(), sessionID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", SpawnToolName, err)
	}
	if len(tr.Turns) == 0 {
		return "", fmt.Errorf("%s: sub-agent %s finished but left no record", SpawnToolName, sessionID)
	}
	return formatTranscript(sessionID, tr, false), nil
}

// formatTranscript renders a finished sub-agent transcript. Error turns (a
// single step with finish_reason "error") render as a failure notice;
// otherwise the final answer text is returned, with steps included on request.
func formatTranscript(sessionID string, tr *store.Transcript, includeSteps bool) string {
	last := tr.Turns[len(tr.Turns)-1]
	if n := len(last.Steps); n > 0 && last.Steps[n-1].Step.FinishReason == errFinishReason {
		text := transcriptErrorText(last)
		if text == "" {
			text = "unknown error"
		}
		return fmt.Sprintf("sub-agent %s failed: %s", sessionID, text)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "sub-agent %s (status: done)\n", sessionID)
	fmt.Fprintf(&b, "turns: %d, steps: %d\n", len(tr.Turns), countSteps(tr.Turns))

	answer := transcriptAnswer(tr.Turns)
	if len(answer) > defaultCellLen {
		answer = answer[:defaultCellLen] + "..."
	}
	fmt.Fprintf(&b, "\n%s\n", answer)

	if includeSteps {
		for _, tt := range tr.Turns {
			for _, st := range tt.Steps {
				fmt.Fprintf(&b, "\n-- step %d (%s) --\n", st.Step.Number, st.Step.FinishReason)
				b.WriteString(stepDetails(st))
			}
		}
	}
	return b.String()
}

// transcriptAnswer concatenates the assistant text across a transcript's
// steps (the sub-agent's final answer and any intermediate narration).
func transcriptAnswer(turns []store.TurnTranscript) string {
	var b strings.Builder
	for _, tt := range turns {
		for _, st := range tt.Steps {
			for _, m := range st.Messages {
				if m.Role != "assistant" {
					continue
				}
				for _, p := range m.Parts {
					if p.Type == "text" {
						if text := strings.TrimSpace(p.Text); text != "" {
							b.WriteString(text)
							b.WriteString("\n")
						}
					}
				}
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// transcriptErrorText extracts the failure text from an error turn.
func transcriptErrorText(tt store.TurnTranscript) string {
	for _, st := range tt.Steps {
		for _, m := range st.Messages {
			if m.Role != "assistant" {
				continue
			}
			for _, p := range m.Parts {
				if p.Type == "text" && strings.TrimSpace(p.Text) != "" {
					return strings.TrimSpace(p.Text)
				}
			}
		}
	}
	return ""
}

// stepDetails renders one step's narration and tool calls with their results.
func stepDetails(st store.StepTranscript) string {
	var b strings.Builder
	for _, m := range st.Messages {
		switch m.Role {
		case "assistant":
			for _, p := range m.Parts {
				switch p.Type {
				case "text":
					if t := strings.TrimSpace(p.Text); t != "" {
						fmt.Fprintf(&b, "  %s\n", truncate(t, defaultCellLen))
					}
				case "tool-call":
					fmt.Fprintf(&b, "  called %s input=%s\n", p.ToolName, truncate(p.ToolInput, defaultCellLen))
				}
			}
		case "tool":
			for _, p := range m.Parts {
				if p.Type != "tool-result" {
					continue
				}
				out := p.ToolOutput
				if p.ToolError != "" {
					out = "error: " + p.ToolError
				}
				fmt.Fprintf(&b, "  -> %s\n", truncate(out, defaultCellLen))
			}
		}
	}
	return b.String()
}

func countSteps(turns []store.TurnTranscript) int {
	n := 0
	for _, tt := range turns {
		n += len(tt.Steps)
	}
	return n
}

// countMessages counts the steps in a flat message slice: one per assistant
// message.
func countMessages(msgs []store.Message) int {
	n := 0
	for _, m := range msgs {
		if m.Role == "assistant" {
			n++
		}
	}
	return n
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (r *Runner) track(sessionID string) chan struct{} {
	done := make(chan struct{})
	r.mu.Lock()
	r.inflight[sessionID] = done
	r.mu.Unlock()
	return done
}

func (r *Runner) untrack(sessionID string) {
	r.mu.Lock()
	if done, ok := r.inflight[sessionID]; ok {
		delete(r.inflight, sessionID)
		close(done)
	}
	r.mu.Unlock()
}

// toolsToStore renders the sub-agent's tool set for its session row.
func toolsToStore(ts []goai.Tool) []store.Tool {
	out := make([]store.Tool, 0, len(ts))
	for _, t := range ts {
		out = append(out, store.Tool{
			Name:                t.Name,
			Description:         t.Description,
			InputSchema:         string(t.InputSchema),
			ProviderDefinedType: t.ProviderDefinedType,
		})
	}
	return out
}
