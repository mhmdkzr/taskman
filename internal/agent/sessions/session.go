package sessions

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/openai"

	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
)

// Create resolves agentName to its model and prompt template, renders the
// system prompt with params, and persists a new session with an empty turn
// history. parentSessionID is nil for a top-level, user-initiated session.
func Create(
	ctx context.Context,
	st *store.Store,
	cfg config.ProviderConfig,
	agentName string,
	params map[string]any,
	parentSessionID *SessionID,
) (SessionID, error) {
	agent, err := AgentByName(ctx, st, agentName)
	if err != nil {
		return SessionID{}, fmt.Errorf("create session: resolve agent: %w", err)
	}

	sysPrompt, err := renderPrompt(agent.TemplateBody, params)
	if err != nil {
		return SessionID{}, fmt.Errorf("create session: render prompt: %w", err)
	}

	providerOpts := map[string]any{
		"reasoning_effort": cfg.ReasoningEffort,
		"useResponsesAPI":  !strings.Contains(strings.TrimRight(cfg.BaseURL, "/"), "/go/v1"),
		"store":            false,
	}

	id, err := createSession(ctx, st.RW(), agent.AgentID, agent.ModelID, sysPrompt, providerOpts, parentSessionID)
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
		providerOptions["useResponsesAPI"] = !strings.Contains(strings.TrimRight(cfg.BaseURL, "/"), "/go/v1")
	}
	if _, ok := providerOptions["store"]; !ok {
		providerOptions["store"] = false
	}

	opts := []goai.Option{
		goai.WithSystem(stored.SystemPrompt),
		goai.WithTools(tools...),
		goai.WithMaxSteps(4),
		goai.WithProviderOptions(providerOptions),
		goai.WithMessages(msgs...),
	}

	result, err := goai.GenerateText(ctx, model, opts...)
	if err != nil {
		return nil, fmt.Errorf("generate text: %w", err)
	}

	if err := appendTurn(ctx, st.RW(), id, prompt, result); err != nil {
		return nil, fmt.Errorf("persist session turn: %w", err)
	}

	return result, nil
}
