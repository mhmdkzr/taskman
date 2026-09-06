// Package agent manages configured agent sessions and tool execution.
package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/app/config"
	"github.com/mhmdkzr/loop/internal/store"
)

// StartSession creates a new, top-level session for the named agent — no
// parent session, unlike a dispatched subagent. params renders the agent's
// prompt template into its system prompt, same as any other agent.
// overrides substitutes the agent's own model and/or reasoning effort for
// this one session, when set (see sessions.Overrides).
func StartSession(
	ctx context.Context,
	st *store.Store,
	cfg config.ProviderConfig,
	agentName string,
	params map[string]any,
	overrides sessions.Overrides,
) (sessions.SessionID, error) {
	id, err := sessions.Create(ctx, st, cfg, agentName, params, nil, overrides)
	if err != nil {
		return sessions.SessionID{}, fmt.Errorf("start session: %w", err)
	}
	return id, nil
}

// ListAgentNames returns every configured agent's name, alphabetically.
func ListAgentNames(ctx context.Context, st *store.Store) ([]string, error) {
	names, err := listAgentNames(ctx, st.RO())
	if err != nil {
		return nil, fmt.Errorf("list agent names: %w", err)
	}
	return names, nil
}

// CreateAgent creates a named agent using the active configured model and
// grants it every registered tool.
func CreateAgent(ctx context.Context, st *store.Store, cfg config.ProviderConfig, name, prompt string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("create agent: name is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("create agent: prompt is required")
	}
	if err := createAgent(ctx, st.RW(), cfg.Model, name, prompt, toolSeeds()); err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	return nil
}

// Respond runs one turn of a top-level session: message is what the user
// sent. Like a dispatched subagent, its tool list comes from its agent_tools
// rows — the same Registry.Resolve every agent uses, not a special-cased
// "every tool" path. A top-level agent currently happens to have every tool
// linked, but which tools that is stays entirely data-driven: removing one
// later is an agent_tools row change, not a code change, and it can still
// stay available to other agents.
func Respond(
	ctx context.Context,
	deps tools.Deps,
	sessionID sessions.SessionID,
	message string,
) (*goai.TextResult, error) {
	deps.SessionID = sessionID
	agentID, err := sessions.AgentIDFor(ctx, deps.Store, sessionID)
	if err != nil {
		return nil, fmt.Errorf("respond: %w", err)
	}
	toolNames, err := sessions.ToolNamesForAgent(ctx, deps.Store, agentID)
	if err != nil {
		return nil, fmt.Errorf("respond: %w", err)
	}
	resolved, err := Tools().Resolve(toolNames, deps)
	if err != nil {
		return nil, fmt.Errorf("respond: %w", err)
	}

	result, err := sessions.Run(ctx, deps.Store, deps.Config.Provider, sessionID, message, resolved)
	if err != nil {
		return nil, fmt.Errorf("respond: %w", err)
	}
	return result, nil
}
