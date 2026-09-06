package sessions

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"text/template"
	"uuid"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/openai"

	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
)

// maxAutoSteps bounds how many model/tool-call rounds a single turn can run
// through on its own (see goai.WithMaxSteps in Run) before it must return
// control to the user - generous enough for a real multi-step task, but not
// unbounded.
const maxAutoSteps = 50

// Overrides lets a caller override what an agent's own configuration would
// otherwise fully determine when creating a session: which model it runs on,
// and how hard it reasons. Both are optional - the zero value falls back to
// the agent's configured model and cfg.ReasoningEffort respectively - so a
// session's model and reasoning effort are parameters per call (settable per
// task, per dispatch, whatever the caller has in scope), not fixed for every
// session a given agent ever runs.
type Overrides struct {
	// Model is a model name (models.model_name, e.g. "claude-sonnet-5"); ""
	// uses the agent's own configured model.
	Model string
	// ReasoningEffort is a raw provider-specific value (e.g. "low", "medium",
	// "high"); "" uses cfg.ReasoningEffort.
	ReasoningEffort string
}

// Create resolves agentName to its model and prompt template, renders the
// system prompt with params, and persists a new session with an empty turn
// history. parentSessionID is nil for a top-level, user-initiated session.
// overrides substitutes the agent's own model and/or the configured
// reasoning effort for this one session, when set.
func Create(
	ctx context.Context,
	st *store.Store,
	cfg config.ProviderConfig,
	agentName string,
	params map[string]any,
	parentSessionID *SessionID,
	overrides Overrides,
) (SessionID, error) {
	agent, err := AgentByName(ctx, st, agentName)
	if err != nil {
		return SessionID{}, fmt.Errorf("create session: resolve agent: %w", err)
	}

	modelID := agent.ModelID
	if overrides.Model != "" {
		modelID, err = ModelIDByName(ctx, st.RO(), overrides.Model)
		if err != nil {
			return SessionID{}, fmt.Errorf("create session: resolve model override: %w", err)
		}
	}

	sysPrompt, err := renderPrompt(agent.TemplateBody, params)
	if err != nil {
		return SessionID{}, fmt.Errorf("create session: render prompt: %w", err)
	}

	reasoningEffort := cfg.ReasoningEffort
	if overrides.ReasoningEffort != "" {
		reasoningEffort = overrides.ReasoningEffort
	}
	providerOpts := map[string]any{
		"reasoning_effort": reasoningEffort,
		"useResponsesAPI":  !strings.Contains(strings.TrimRight(cfg.BaseURL, "/"), "/go/v1"),
		"store":            false,
	}

	id, err := createSession(
		ctx,
		st.RW(),
		agent.AgentID,
		modelID,
		sysPrompt,
		providerOpts,
		parentSessionID,
	)
	if err != nil {
		return SessionID{}, fmt.Errorf("create session: %w", err)
	}
	return id, nil
}

// renderPrompt executes an agent's prompt template with params.
func renderPrompt(body string, params map[string]any) (string, error) {
	tmpl, err := template.New("prompt").Parse(body)
	if err != nil {
		return "", fmt.Errorf("parse prompt template: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("execute prompt template: %w", err)
	}
	return buf.String(), nil
}

func Run(
	ctx context.Context,
	st *store.Store,
	cfg config.ProviderConfig,
	id SessionID,
	prompt string,
	tools []goai.Tool,
) (*goai.TextResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("prompt is empty")
	}

	stored, err := sessionByID(ctx, st.RO(), id)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	msgs := make([]provider.Message, 0, len(stored.Turns)*2+1)
	for _, turn := range stored.Turns {
		msgs = append(msgs, goai.UserMessage(turn.Prompt))
		if turn.Result != nil {
			msgs = append(msgs, turn.Result.ResponseMessages...)
		}
	}
	msgs = append(msgs, goai.UserMessage(prompt))

	modelName, err := modelNameByID(ctx, st.RO(), stored.ModelID)
	if err != nil {
		return nil, fmt.Errorf("resolve session model: %w", err)
	}
	model := openai.Chat(
		modelName,
		openai.WithBaseURL(cfg.BaseURL),
		openai.WithAPIKey(cfg.APIKeyOpenCode),
	)
	providerOptions := stored.ProviderOptions
	if _, ok := providerOptions["useResponsesAPI"]; !ok {
		if providerOptions == nil {
			providerOptions = make(map[string]any)
		}
		providerOptions["useResponsesAPI"] = !strings.Contains(
			strings.TrimRight(cfg.BaseURL, "/"),
			"/go/v1",
		)
	}
	if _, ok := providerOptions["store"]; !ok {
		providerOptions["store"] = false
	}

	// turnID and startTurn are the write-ahead marker: the row exists (as
	// turnStatusRunning) before GenerateText's model/tool-call loop begins, so
	// a crash mid-loop leaves a detectable, reconcilable row instead of no
	// trace at all (see ReconcileInterrupted). The hooks below persist each
	// step/tool call as it happens, not just the final result, so a
	// reconciliation pass has real progress to replay rather than an empty log.
	turnID := uuid.NewV7()
	if err := startTurn(ctx, st.RW(), id, turnID, prompt); err != nil {
		return nil, fmt.Errorf("start session turn: %w", err)
	}

	var seq atomic.Int64
	nextSeq := func() int64 { return seq.Add(1) }
	recordEvent := func(eventType string, payload any) {
		if err := recordTurnEvent(ctx, st.RW(), turnID, nextSeq(), eventType, payload); err != nil {
			// Not fatal to the tool loop: losing one event only narrows what a
			// future crash-reconciliation pass can replay for this turn, it
			// does not affect the live in-memory result. Must still be logged
			// per repo convention (no silent errors).
			slog.Error(
				"record session turn event",
				"error",
				err,
				"turn_id",
				turnID.String(),
				"event_type",
				eventType,
			)
		}
	}

	opts := []goai.Option{
		goai.WithSystem(stored.SystemPrompt),
		goai.WithTools(tools...),
		goai.WithProviderOptions(providerOptions),
		goai.WithMessages(msgs...),
		goai.WithHeaders(map[string]string{"X-Opencode-Session": id.String()}), // TODO: make it conditional, only set when provider is opencode
		goai.WithPromptCaching(true),
		goai.WithMaxRetries(10),
		// goai's own default (1) runs at most one round of tool calls per
		// turn and stops - it will not see a tool's result and decide to act
		// on it. maxAutoSteps lets a turn keep looping (model response ->
		// tool calls -> tool results -> next model response -> ...) on its
		// own until it produces a final answer with no further tool calls,
		// or this cap is hit.
		goai.WithMaxSteps(maxAutoSteps),
		goai.WithOnStepFinish(func(step goai.StepResult) {
			payload := stepFinishPayload{
				Step:      step.Number,
				Text:      step.Text,
				Reasoning: step.Reasoning,
			}
			for _, tc := range step.ToolCalls {
				payload.ToolCalls = append(
					payload.ToolCalls,
					turnEventToolCall{ID: tc.ID, Name: tc.Name, Input: tc.Input},
				)
			}
			recordEvent("step_finish", payload)
		}),
		goai.WithOnToolCallStart(func(info goai.ToolCallStartInfo) {
			recordEvent("tool_call_start", toolCallStartPayload{
				Step: info.Step, ToolCallID: info.ToolCallID, ToolName: info.ToolName,
			})
		}),
		goai.WithOnToolCall(func(info goai.ToolCallInfo) {
			payload := toolCallResultPayload{
				Step: info.Step, ToolCallID: info.ToolCallID, ToolName: info.ToolName, Output: info.Output,
			}
			if info.Error != nil {
				payload.Error = info.Error.Error()
			}
			recordEvent("tool_call_result", payload)
		}),
	}

	result, genErr := goai.GenerateText(ctx, model, opts...)
	if genErr != nil {
		// The loop didn't return normally, but this process is still alive -
		// close the turn out now from whatever events the hooks above already
		// recorded, rather than leaving it turnStatusRunning until a future
		// boot's ReconcileInterrupted trips over it. Same synthesis path as
		// crash recovery: every requested tool call still gets a real or
		// synthesized result, so the stored conversation stays replayable.
		if err := reconcileInterruptedTurn(ctx, st.RW(), turnID); err != nil {
			slog.Error("close failed session turn", "error", err, "turn_id", turnID.String())
		}
		return nil, fmt.Errorf("generate text: %w", genErr)
	}

	if err := completeTurn(ctx, st.RW(), id, turnID, result); err != nil {
		return nil, fmt.Errorf("persist session turn: %w", err)
	}

	return result, nil
}
