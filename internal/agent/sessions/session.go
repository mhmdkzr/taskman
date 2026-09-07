package sessions

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
)

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
