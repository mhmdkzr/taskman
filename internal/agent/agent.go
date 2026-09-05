// Package agent manages configured agent sessions and tool execution.
package agent

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools"
	"github.com/mhmdkzr/loop/internal/app/config"
)

// StartSession creates a new, top-level session for the named agent — no
// parent session, unlike a dispatched subagent. params renders the agent's
// prompt template into its system prompt, same as any other agent.
func StartSession(
	ctx context.Context,
	db *sql.DB,
	cfg config.ProviderConfig,
	agentName string,
	params map[string]any,
) (sessions.SessionID, error) {
	id, err := sessions.Create(ctx, db, cfg, agentName, params, nil)
	if err != nil {
		return sessions.SessionID{}, fmt.Errorf("start session: %w", err)
	}
	return id, nil
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
	db *sql.DB,
	cfg config.ProviderConfig,
	sessionID sessions.SessionID,
	message string,
) (*goai.TextResult, error) {
	agentID, err := sessions.AgentIDFor(ctx, db, sessionID)
	if err != nil {
		return nil, fmt.Errorf("respond: %w", err)
	}
	toolNames, err := sessions.ToolNamesForAgent(ctx, db, agentID)
	if err != nil {
		return nil, fmt.Errorf("respond: %w", err)
	}
	deps := tools.Deps{DB: db, SessionID: sessionID}
	resolved, err := Tools().Resolve(toolNames, deps)
	if err != nil {
		return nil, fmt.Errorf("respond: %w", err)
	}

	result, err := sessions.Run(ctx, db, cfg, sessionID, message, resolved)
	if err != nil {
		return nil, fmt.Errorf("respond: %w", err)
	}
	return result, nil
}
