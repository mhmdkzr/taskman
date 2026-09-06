package mcp

import (
	"context"
	"fmt"
	"strings"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	toolrepo "github.com/mhmdkzr/loop/internal/agent/tools/tool"
	"github.com/mhmdkzr/loop/internal/app"
)

// addToolTools registers read-only lookups over loop's registered tools
// (what an agent can be granted, not a way to invoke them).
func addToolTools(s *gomcp.Server, a app.App) {
	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "tool_list",
		Description: "List every tool registered in loop, across all agents.",
	}, toolListHandler(a))

	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "tool_get",
		Description: "Get a registered tool's description and schema, by name.",
	}, toolGetHandler(a))
}

// mcpTool mirrors toolrepo.Tool with its ID as a plain string - see the
// comment on mcpTask in task.go for why toolrepo.Tool.ID isn't used directly.
type mcpTool struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	InputSchema  string `json:"input_schema"`
	OutputSchema string `json:"output_schema"`
}

func newMCPTool(t toolrepo.Tool) mcpTool {
	return mcpTool{
		ID: t.ID.String(), Name: t.Name, Description: t.Description,
		InputSchema: t.InputSchema, OutputSchema: t.OutputSchema,
	}
}

type toolListInput struct{}

type toolListOutput struct {
	Tools []mcpTool `json:"tools"`
}

func toolListHandler(a app.App) gomcp.ToolHandlerFor[toolListInput, toolListOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, _ toolListInput) (*gomcp.CallToolResult, toolListOutput, error) {
		tools, err := toolrepo.ListTools(ctx, a.Deps.Store.RO())
		if err != nil {
			return nil, toolListOutput{}, fmt.Errorf("list tools: %w", err)
		}
		out := toolListOutput{Tools: make([]mcpTool, len(tools))}
		for i, t := range tools {
			out.Tools[i] = newMCPTool(t)
		}
		return nil, out, nil
	}
}

type toolGetInput struct {
	Name string `json:"name" jsonschema:"Registered tool_name (see tool_list)."`
}

type toolGetOutput struct {
	Tool mcpTool `json:"tool"`
}

func toolGetHandler(a app.App) gomcp.ToolHandlerFor[toolGetInput, toolGetOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in toolGetInput) (*gomcp.CallToolResult, toolGetOutput, error) {
		if strings.TrimSpace(in.Name) == "" {
			return nil, toolGetOutput{}, fmt.Errorf("name is required")
		}
		t, err := toolrepo.GetToolByName(ctx, a.Deps.Store.RO(), in.Name)
		if err != nil {
			return nil, toolGetOutput{}, fmt.Errorf("get tool: %w", err)
		}
		return nil, toolGetOutput{Tool: newMCPTool(t)}, nil
	}
}
