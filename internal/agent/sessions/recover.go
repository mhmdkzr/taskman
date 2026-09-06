package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"

	"github.com/mhmdkzr/loop/internal/store"
)

// errInterruptedNotAttempted marks a tool call the model requested but that
// never even started before the process crashed.
var errInterruptedNotAttempted = errors.New(
	"interrupted: process terminated before this tool call was ever attempted",
)

// errInterruptedUnknownOutcome marks a tool call that started but whose
// result was never recorded before the process crashed - it may have
// partially or fully executed its side effects.
var errInterruptedUnknownOutcome = errors.New(
	"interrupted: process terminated after this tool call started; its outcome is unknown - verify current state before retrying, especially for non-idempotent actions",
)

// turnEventToolCall is the tool-call shape shared by the step_finish event
// payload and provider.ToolCall/PartToolCall.
type turnEventToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input,omitempty"`
}

// stepFinishPayload is the session_turn_events payload for a "step_finish"
// event, recorded from goai's OnStepFinish hook.
type stepFinishPayload struct {
	Step      int                 `json:"step"`
	Text      string              `json:"text"`
	Reasoning string              `json:"reasoning,omitempty"`
	ToolCalls []turnEventToolCall `json:"tool_calls,omitempty"`
}

// toolCallStartPayload is the session_turn_events payload for a
// "tool_call_start" event, recorded from goai's OnToolCallStart hook.
type toolCallStartPayload struct {
	Step       int    `json:"step"`
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
}

// toolCallResultPayload is the session_turn_events payload for a
// "tool_call_result" event, recorded from goai's OnToolCall hook.
type toolCallResultPayload struct {
	Step       int    `json:"step"`
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
}

// ReconcileInterrupted closes out every turn left in turnStatusRunning across
// every session - the write-ahead marker inserted by startTurn before a
// model/tool-call loop begins, updated to turnStatusCompleted by completeTurn
// once it returns normally. A row still turnStatusRunning is, by construction,
// impossible to observe from a live process still inside that same turn's
// Run call, so at process start any such row was orphaned by a crash.
//
// Call this once at boot, before any session is resumed, so a crash is
// detected and repaired proactively rather than left for the next Run call
// (or never, if that session is never used again) to trip over. It returns
// the number of turns reconciled.
func ReconcileInterrupted(ctx context.Context, st *store.Store) (int, error) {
	turnIDs, err := runningTurnIDs(ctx, st.RO())
	if err != nil {
		return 0, fmt.Errorf("reconcile interrupted turns: %w", err)
	}
	for _, turnID := range turnIDs {
		if err := reconcileInterruptedTurn(ctx, st.RW(), turnID); err != nil {
			return 0, fmt.Errorf("reconcile interrupted turn %s: %w", turnID, err)
		}
		slog.Warn("recovered turn interrupted by a crash", "turn_id", turnID.String())
	}
	return len(turnIDs), nil
}

// runningTurnIDs lists every session_turns row still in turnStatusRunning.
func runningTurnIDs(ctx context.Context, db *sql.DB) ([]uuid.UUID, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT turn_id, session_id FROM session_turns WHERE status = ?`,
		string(turnStatusRunning))
	if err != nil {
		return nil, fmt.Errorf("query running turns: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close running turn rows", "error", err)
		}
	}()

	var ids []uuid.UUID
	for rows.Next() {
		var turnIDStr, sessionIDStr string
		if err := rows.Scan(&turnIDStr, &sessionIDStr); err != nil {
			return nil, fmt.Errorf("scan running turn: %w", err)
		}
		turnID, err := uuid.Parse(turnIDStr)
		if err != nil {
			return nil, fmt.Errorf("parse turn id: %w", err)
		}
		ids = append(ids, turnID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate running turns: %w", err)
	}
	return ids, nil
}

// reconcileInterruptedTurn replays turnID's session_turn_events into a
// synthesized partial goai.TextResult and closes the row out as
// turnStatusInterrupted with that result. Every tool call the model
// requested gets a tool-result message - real if one was recorded, or a
// synthetic "interrupted" one otherwise - so the stored conversation is
// always valid to replay back into a future model call: a dangling tool
// call with no result is an invalid message sequence for every provider.
func reconcileInterruptedTurn(ctx context.Context, db *sql.DB, turnID uuid.UUID) error {
	result, err := synthesizeInterruptedResult(ctx, db, turnID)
	if err != nil {
		return fmt.Errorf("synthesize result: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal synthesized result: %w", err)
	}

	res, err := db.ExecContext(
		ctx,
		`
		UPDATE session_turns
		SET result = ?, status = ?, completed_at = ?
		WHERE turn_id = ? AND status = ?`,
		string(
			resultJSON,
		),
		string(turnStatusInterrupted),
		time.Now().UTC().Format(time.RFC3339Nano),
		turnID.String(),
		string(turnStatusRunning),
	)
	if err != nil {
		return fmt.Errorf("update session turn: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("no running turn %s to reconcile", turnID)
	}
	return nil
}

// synthesizeInterruptedResult replays turnID's events (in seq order) into a
// goai.TextResult: one assistant message per recorded step_finish, followed
// by one tool-result message per tool call that step's model response
// requested - matched against tool_call_result events when available, and
// synthesized as an error result (distinguishing "never attempted" from
// "started but outcome unknown") otherwise.
func synthesizeInterruptedResult(
	ctx context.Context,
	db *sql.DB,
	turnID uuid.UUID,
) (*goai.TextResult, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT event_type, payload
		FROM session_turn_events
		WHERE turn_id = ?
		ORDER BY seq, event_id`, turnID.String())
	if err != nil {
		return nil, fmt.Errorf("query turn events: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close turn event rows", "error", err)
		}
	}()

	var steps []stepFinishPayload
	started := make(map[string]bool)
	results := make(map[string]toolCallResultPayload)
	for rows.Next() {
		var eventType, payloadRaw string
		if err := rows.Scan(&eventType, &payloadRaw); err != nil {
			return nil, fmt.Errorf("scan turn event: %w", err)
		}
		switch eventType {
		case "step_finish":
			var p stepFinishPayload
			if err := json.Unmarshal([]byte(payloadRaw), &p); err != nil {
				return nil, fmt.Errorf("decode step_finish payload: %w", err)
			}
			steps = append(steps, p)
		case "tool_call_start":
			var p toolCallStartPayload
			if err := json.Unmarshal([]byte(payloadRaw), &p); err != nil {
				return nil, fmt.Errorf("decode tool_call_start payload: %w", err)
			}
			started[p.ToolCallID] = true
		case "tool_call_result":
			var p toolCallResultPayload
			if err := json.Unmarshal([]byte(payloadRaw), &p); err != nil {
				return nil, fmt.Errorf("decode tool_call_result payload: %w", err)
			}
			results[p.ToolCallID] = p
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate turn events: %w", err)
	}

	result := &goai.TextResult{}
	for _, step := range steps {
		var assistantParts []provider.Part
		if step.Text != "" {
			assistantParts = append(
				assistantParts,
				provider.Part{Type: provider.PartText, Text: step.Text},
			)
			result.Text = step.Text
		}
		if step.Reasoning != "" {
			assistantParts = append(
				assistantParts,
				provider.Part{Type: provider.PartReasoning, Text: step.Reasoning},
			)
		}
		for _, tc := range step.ToolCalls {
			assistantParts = append(assistantParts, provider.Part{
				Type: provider.PartToolCall, ToolCallID: tc.ID, ToolName: tc.Name, ToolInput: tc.Input,
			})
		}
		if len(assistantParts) > 0 {
			result.ResponseMessages = append(result.ResponseMessages,
				provider.Message{Role: provider.RoleAssistant, Content: assistantParts})
		}

		var toolCalls []provider.ToolCall
		var toolResults []provider.ToolResult
		for _, tc := range step.ToolCalls {
			toolCalls = append(
				toolCalls,
				provider.ToolCall{ID: tc.ID, Name: tc.Name, Input: tc.Input},
			)
			tr := toolResultFor(tc, started, results)
			toolResults = append(toolResults, tr)
			result.ResponseMessages = append(result.ResponseMessages,
				goai.ToolMessage(tc.ID, tc.Name, tr.Output))
		}

		result.Steps = append(result.Steps, goai.StepResult{
			Number: step.Step, Text: step.Text, Reasoning: step.Reasoning,
			ToolCalls: toolCalls, ToolResults: toolResults,
		})
		result.ToolCalls = toolCalls
	}
	return result, nil
}

// toolResultFor resolves the outcome of one requested tool call: the real
// recorded result if the tool_call_result event exists, or a synthesized
// error distinguishing "never attempted" (no tool_call_start event either)
// from "started but outcome unknown" (started, crashed before finishing).
func toolResultFor(
	tc turnEventToolCall,
	started map[string]bool,
	results map[string]toolCallResultPayload,
) provider.ToolResult {
	if res, ok := results[tc.ID]; ok {
		if res.Error != "" {
			return provider.ToolResult{
				ToolCallID: tc.ID, ToolName: tc.Name,
				Output: "error: " + res.Error, Error: errors.New(res.Error), IsError: true,
			}
		}
		return provider.ToolResult{ToolCallID: tc.ID, ToolName: tc.Name, Output: res.Output}
	}
	cause := errInterruptedNotAttempted
	if started[tc.ID] {
		cause = errInterruptedUnknownOutcome
	}
	return provider.ToolResult{
		ToolCallID: tc.ID, ToolName: tc.Name,
		Output: "error: " + cause.Error(), Error: cause, IsError: true,
	}
}
