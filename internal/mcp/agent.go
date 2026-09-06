package mcp

import (
	"context"
	"fmt"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/loop/internal/agent"
	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/app"
)

// addAgentTools registers read-only lookups over loop's configured agents -
// what session_create needs an agent_name for.
func addAgentTools(s *gomcp.Server, a app.App) {
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "agent_list",
		Description: "List every loop agent's name.",
	}, agentListHandler(a))

	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "agent_get",
		Description: "Get a loop agent's tools, by name.",
	}, agentGetHandler(a))
}

type agentListInput struct{}

type agentListOutput struct {
	AgentNames []string `json:"agent_names"`
}

func agentListHandler(a app.App) gomcp.ToolHandlerFor[agentListInput, agentListOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, _ agentListInput) (*gomcp.CallToolResult, agentListOutput, error) {
		names, err := agent.ListAgentNames(ctx, a.Deps.Store)
		if err != nil {
			return nil, agentListOutput{}, fmt.Errorf("list agents: %w", err)
		}
		return nil, agentListOutput{AgentNames: names}, nil
	}
}

type agentGetInput struct {
	Name string `json:"name" jsonschema:"The agent's name (see agent_list)."`
}

type agentGetOutput struct {
	AgentName string   `json:"agent_name"`
	ToolNames []string `json:"tool_names"`
}

func agentGetHandler(a app.App) gomcp.ToolHandlerFor[agentGetInput, agentGetOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in agentGetInput) (*gomcp.CallToolResult, agentGetOutput, error) {
		if strings.TrimSpace(in.Name) == "" {
			return nil, agentGetOutput{}, fmt.Errorf("name is required")
		}
		cfg, err := sessions.AgentByName(ctx, a.Deps.Store, in.Name)
		if err != nil {
			return nil, agentGetOutput{}, fmt.Errorf("get agent: %w", err)
		}
		toolNames, err := sessions.ToolNamesForAgent(ctx, a.Deps.Store, cfg.AgentID)
		if err != nil {
			return nil, agentGetOutput{}, fmt.Errorf("get agent tools: %w", err)
		}
		return nil, agentGetOutput{AgentName: in.Name, ToolNames: toolNames}, nil
	}
}
