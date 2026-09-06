package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"uuid"

	gomcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mhmdkzr/loop/internal/agent"
	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools/ask"
	"github.com/mhmdkzr/loop/internal/app"
)

// addSessionTools registers the one read/write surface this package exposes:
// creating and prompting loop's own agent sessions, and answering a question
// one asks back. Every other tool group registered from server.go is
// read-only.
func addSessionTools(s *gomcp.Server, a app.App) {
	gomcp.AddTool(s, &gomcp.Tool{
		Name: "session_create",
		Description: "Start a new top-level loop agent session and send it its first message. " +
			"Returns immediately once the turn has started - it does not wait for the agent to " +
			"finish, since that can take a while and the agent may ask you something first. " +
			"Poll session_get with the returned session_id to see progress and read the reply.",
	}, sessionCreateHandler(a))

	gomcp.AddTool(s, &gomcp.Tool{
		Name: "session_respond",
		Description: "Send the next message to an existing loop agent session. Returns " +
			"immediately once the turn has started, for the same reason as session_create - " +
			"poll session_get to see progress and read the reply.",
	}, sessionRespondHandler(a))

	gomcp.AddTool(s, &gomcp.Tool{
		Name: "session_answer_ask",
		Description: "Answer a question a loop agent session asked back mid-turn (see the " +
			"pending_ask field from session_get). Answering unblocks that turn, which then " +
			"continues on its own - poll session_get again to see what it does next.",
	}, sessionAnswerAskHandler(a))

	gomcp.AddTool(s, &gomcp.Tool{
		Name: "session_get",
		Description: "Read a loop agent session's full detail: its agent, status, every turn " +
			"(prompt, reasoning steps, tool calls, reply), and - if the current turn is waiting " +
			"on one - the pending question to answer via session_answer_ask.",
	}, sessionGetHandler(a))

	gomcp.AddTool(s, &gomcp.Tool{
		Name:        "session_list",
		Description: "List every loop agent session, newest first, with a short summary of each.",
	}, sessionListHandler(a))
}

// sessionCreateInput is the input for session_create.
type sessionCreateInput struct {
	AgentName string `json:"agent_name" jsonschema:"The loop agent to run as (see agent_list)."`
	Prompt    string `json:"prompt"     jsonschema:"The first message to send it."`
}

// sessionStartedOutput is returned by session_create and session_respond:
// the turn has started but not necessarily finished - see session_get.
type sessionStartedOutput struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

func sessionCreateHandler(a app.App) gomcp.ToolHandlerFor[sessionCreateInput, sessionStartedOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in sessionCreateInput) (*gomcp.CallToolResult, sessionStartedOutput, error) {
		if strings.TrimSpace(in.AgentName) == "" {
			return nil, sessionStartedOutput{}, fmt.Errorf("agent_name is required")
		}
		if strings.TrimSpace(in.Prompt) == "" {
			return nil, sessionStartedOutput{}, fmt.Errorf("prompt is required")
		}

		id, err := agent.StartSession(ctx, a.Deps.Store, a.Cfg.Provider, in.AgentName, nil)
		if err != nil {
			return nil, sessionStartedOutput{}, fmt.Errorf("start session: %w", err)
		}
		runTurnInBackground(a, id, in.Prompt) //nolint:contextcheck // detached by design, see below
		return nil, sessionStartedOutput{SessionID: id.String(), Status: "running"}, nil
	}
}

// sessionRespondInput is the input for session_respond.
type sessionRespondInput struct {
	SessionID string `json:"session_id" jsonschema:"The session to send this message to."`
	Prompt    string `json:"prompt"     jsonschema:"The message to send it."`
}

func sessionRespondHandler(a app.App) gomcp.ToolHandlerFor[sessionRespondInput, sessionStartedOutput] {
	return func(_ context.Context, _ *gomcp.CallToolRequest, in sessionRespondInput) (*gomcp.CallToolResult, sessionStartedOutput, error) {
		id, err := parseSessionID(in.SessionID)
		if err != nil {
			return nil, sessionStartedOutput{}, err
		}
		if strings.TrimSpace(in.Prompt) == "" {
			return nil, sessionStartedOutput{}, fmt.Errorf("prompt is required")
		}
		runTurnInBackground(a, id, in.Prompt) //nolint:contextcheck // detached by design, see below
		return nil, sessionStartedOutput{SessionID: id.String(), Status: "running"}, nil
	}
}

// runTurnInBackground starts a turn without blocking the MCP tool call that
// requested it: agent.Respond can take a while, and may itself block on an
// ask tool call the caller can only answer via a separate session_answer_ask
// call - which it could never make if this tool call were still holding the
// only "turn" of the conversation. It deliberately does not take the
// request's context (see context.WithoutCancel below) so the turn keeps
// running after this handler returns and the MCP response is sent.
func runTurnInBackground(a app.App, id sessions.SessionID, prompt string) {
	runCtx := context.WithoutCancel(context.Background())
	go func() {
		if _, err := agent.Respond(runCtx, a.Deps.AgentTools, id, prompt); err != nil {
			slog.Error("mcp: run session turn", "error", err, "session_id", id.String())
		}
	}()
}

// sessionAnswerAskInput is the input for session_answer_ask.
type sessionAnswerAskInput struct {
	SessionID string `json:"session_id"         jsonschema:"The session the question came from."`
	AskID     string `json:"ask_id"             jsonschema:"The pending_ask.ask_id from session_get."`
	Selected  []int  `json:"selected,omitempty" jsonschema:"IDs of any predefined options chosen."`
	Custom    string `json:"custom,omitempty"   jsonschema:"A free-form answer, instead of or alongside selected."`
}

type okOutput struct {
	Status string `json:"status"`
}

func sessionAnswerAskHandler(a app.App) gomcp.ToolHandlerFor[sessionAnswerAskInput, okOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in sessionAnswerAskInput) (*gomcp.CallToolResult, okOutput, error) {
		// SessionID is only used to scope the tool's inputs to one session in
		// the schema; the ask itself is looked up by ask_id alone.
		if _, err := parseSessionID(in.SessionID); err != nil {
			return nil, okOutput{}, err
		}
		askID, err := uuid.Parse(in.AskID)
		if err != nil {
			return nil, okOutput{}, fmt.Errorf("parse ask_id: %w", err)
		}
		if err := ask.AnswerAsk(ctx, a.Deps.Store.RW(), askID, in.Selected, in.Custom); err != nil {
			return nil, okOutput{}, fmt.Errorf("answer ask: %w", err)
		}
		return nil, okOutput{Status: "ok"}, nil
	}
}

// sessionGetInput is the input for session_get.
type sessionGetInput struct {
	SessionID string `json:"session_id" jsonschema:"The session to read."`
}

// mcpToolCall is one tool call within a turn, for MCP output.
type mcpToolCall struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// mcpTurn is one turn within a session, for MCP output.
type mcpTurn struct {
	Prompt    string        `json:"prompt"`
	Reply     string        `json:"reply"`
	ToolCalls []mcpToolCall `json:"tool_calls,omitempty"`
	Status    string        `json:"status"`
	CreatedAt string        `json:"created_at"`
}

// mcpPendingAsk mirrors ask.PendingAsk for MCP output.
type mcpPendingAsk struct {
	AskID       string         `json:"ask_id"`
	Question    string         `json:"question"`
	Options     []mcpAskOption `json:"options,omitempty"`
	MultiSelect bool           `json:"multi_select"`
}

type mcpAskOption struct {
	ID          int    `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// sessionDetailOutput is the output of session_get.
type sessionDetailOutput struct {
	SessionID       string         `json:"session_id"`
	AgentName       string         `json:"agent_name"`
	ModelName       string         `json:"model_name"`
	ParentSessionID string         `json:"parent_session_id,omitempty"`
	Status          string         `json:"status"`
	Turns           []mcpTurn      `json:"turns"`
	PendingAsk      *mcpPendingAsk `json:"pending_ask,omitempty"`
}

func sessionGetHandler(a app.App) gomcp.ToolHandlerFor[sessionGetInput, sessionDetailOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, in sessionGetInput) (*gomcp.CallToolResult, sessionDetailOutput, error) {
		id, err := parseSessionID(in.SessionID)
		if err != nil {
			return nil, sessionDetailOutput{}, err
		}
		detail, err := sessions.GetSessionDetail(ctx, a.Deps.Store, id)
		if err != nil {
			return nil, sessionDetailOutput{}, fmt.Errorf("get session: %w", err)
		}

		out := sessionDetailOutput{
			SessionID: detail.SessionID.String(),
			AgentName: detail.AgentName,
			ModelName: detail.ModelName,
			Status:    "new",
		}
		if detail.ParentSessionID != nil {
			out.ParentSessionID = detail.ParentSessionID.String()
		}
		out.Turns = make([]mcpTurn, len(detail.Turns))
		for i, t := range detail.Turns {
			out.Status = t.Status
			turn := mcpTurn{Prompt: t.Prompt, Status: t.Status, CreatedAt: t.CreatedAt}
			for _, step := range t.Steps {
				if step.Text != "" {
					if turn.Reply != "" {
						turn.Reply += "\n"
					}
					turn.Reply += step.Text
				}
				for _, tc := range step.ToolCalls {
					turn.ToolCalls = append(turn.ToolCalls, mcpToolCall{
						Name: tc.Name, Input: string(tc.Input), Output: tc.Output, Error: tc.Error,
					})
				}
			}
			out.Turns[i] = turn

			if t.Status == "running" {
				pending, err := ask.PendingAskForTurn(ctx, a.Deps.Store.RO(), t.TurnID)
				if err != nil {
					return nil, sessionDetailOutput{}, fmt.Errorf("get pending ask: %w", err)
				}
				if pending != nil {
					out.PendingAsk = &mcpPendingAsk{
						AskID: pending.AskID.String(), Question: pending.Question, MultiSelect: pending.MultiSelect,
					}
					for _, o := range pending.Options {
						out.PendingAsk.Options = append(out.PendingAsk.Options,
							mcpAskOption{ID: o.ID, Label: o.Label, Description: o.Description})
					}
				}
			}
		}
		return nil, out, nil
	}
}

// sessionListInput is the input for session_list.
type sessionListInput struct{}

// mcpSessionSummary is one session summarized, for MCP output.
type mcpSessionSummary struct {
	SessionID       string `json:"session_id"`
	AgentName       string `json:"agent_name"`
	ParentSessionID string `json:"parent_session_id,omitempty"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
	TurnCount       int    `json:"turn_count"`
	LastPrompt      string `json:"last_prompt,omitempty"`
}

type sessionListOutput struct {
	Sessions []mcpSessionSummary `json:"sessions"`
}

func sessionListHandler(a app.App) gomcp.ToolHandlerFor[sessionListInput, sessionListOutput] {
	return func(ctx context.Context, _ *gomcp.CallToolRequest, _ sessionListInput) (*gomcp.CallToolResult, sessionListOutput, error) {
		summaries, err := sessions.ListSessions(ctx, a.Deps.Store)
		if err != nil {
			return nil, sessionListOutput{}, fmt.Errorf("list sessions: %w", err)
		}
		out := sessionListOutput{Sessions: make([]mcpSessionSummary, len(summaries))}
		for i, s := range summaries {
			row := mcpSessionSummary{
				SessionID: s.SessionID.String(), AgentName: s.AgentName, Status: s.Status,
				CreatedAt: s.CreatedAt, TurnCount: s.TurnCount, LastPrompt: s.LastPrompt,
			}
			if s.ParentSessionID != nil {
				row.ParentSessionID = s.ParentSessionID.String()
			}
			out.Sessions[i] = row
		}
		return nil, out, nil
	}
}

func parseSessionID(raw string) (sessions.SessionID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return sessions.SessionID{}, fmt.Errorf("parse session_id: %w", err)
	}
	return sessions.SessionID(id), nil
}
