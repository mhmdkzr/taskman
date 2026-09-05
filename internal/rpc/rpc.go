// Package rpc exposes the application's operational API over JSON-RPC 2.0.
package rpc

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"uuid"

	"github.com/mhmdkzr/loop/internal/agent"
	"github.com/mhmdkzr/loop/internal/agent/models"
	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/app"
)

const version = "2.0"

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type Handler struct {
	app app.App
}

func NewHandler(a app.App) http.HandlerFunc {
	return (&Handler{app: a}).ServeHTTP
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		write(w, response{JSONRPC: version, ID: nil, Error: &responseError{Code: -32700, Message: "parse error", Data: err.Error()}})
		return
	}
	if req.JSONRPC != version || req.Method == "" {
		write(w, response{JSONRPC: version, ID: req.ID, Error: &responseError{Code: -32600, Message: "invalid request"}})
		return
	}

	result, rpcErr := h.dispatch(r.Context(), req.Method, req.Params)
	resp := response{JSONRPC: version, ID: req.ID, Result: result}
	if rpcErr != nil {
		resp.Result = nil
		resp.Error = rpcErr
	}
	write(w, resp)
}

func (h *Handler) dispatch(ctx context.Context, method string, params json.RawMessage) (any, *responseError) {
	switch method {
	case "system.info":
		return map[string]any{"name": "loop", "rpc": version}, nil
	case "tools.list":
		names := make([]string, 0, len(agent.Tools()))
		for name := range agent.Tools() {
			names = append(names, name)
		}
		sort.Strings(names)
		return names, nil
	case "models.list":
		var output bytes.Buffer
		if err := models.List(ctx, h.app.Deps.DB, &output, true); err != nil {
			return nil, internalError(err)
		}
		var result any
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			return nil, internalError(err)
		}
		return result, nil
	case "models.refresh":
		if err := models.Refresh(ctx, h.app.Deps.DB, h.app.Cfg.Provider); err != nil {
			return nil, internalError(err)
		}
		return map[string]bool{"refreshed": true}, nil
	case "agent.list":
		return listAgents(ctx, h.app.Deps.DB)
	case "agent.create":
		var in struct {
			Name   string `json:"name"`
			Prompt string `json:"prompt"`
		}
		if err := decodeParams(params, &in); err != nil || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Prompt) == "" {
			return nil, invalidParams("name and prompt are required")
		}
		if err := agent.CreateAgent(ctx, h.app.Deps.DB, h.app.Cfg.Provider, in.Name, in.Prompt); err != nil {
			return nil, internalError(err)
		}
		return map[string]string{"name": in.Name}, nil
	case "session.create":
		var in struct {
			Agent  string         `json:"agent"`
			Params map[string]any `json:"params"`
		}
		if err := decodeParams(params, &in); err != nil || strings.TrimSpace(in.Agent) == "" {
			return nil, invalidParams("agent is required")
		}
		id, err := agent.StartSession(ctx, h.app.Deps.DB, h.app.Cfg.Provider, in.Agent, in.Params)
		if err != nil {
			return nil, internalError(err)
		}
		return map[string]string{"session_id": id.String()}, nil
	case "session.respond":
		var in struct {
			SessionID string `json:"session_id"`
			Message   string `json:"message"`
		}
		if err := decodeParams(params, &in); err != nil || strings.TrimSpace(in.SessionID) == "" || strings.TrimSpace(in.Message) == "" {
			return nil, invalidParams("session_id and message are required")
		}
		parsed, err := uuid.Parse(in.SessionID)
		if err != nil {
			return nil, invalidParams("session_id must be a UUID")
		}
		result, err := agent.Respond(ctx, h.app.Deps.AgentTools, sessions.SessionID(parsed), in.Message)
		if err != nil {
			return nil, internalError(err)
		}
		return result, nil
	default:
		return nil, &responseError{Code: -32601, Message: "method not found"}
	}
}

func listAgents(ctx context.Context, db *sql.DB) (any, *responseError) {
	rows, err := db.QueryContext(ctx, `SELECT agent_id, agent_name, model_id FROM agents ORDER BY agent_name`)
	if err != nil {
		return nil, internalError(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close RPC agent rows", "error", err)
		}
	}()
	result := make([]map[string]string, 0)
	for rows.Next() {
		var id, name, model string
		if err := rows.Scan(&id, &name, &model); err != nil {
			return nil, internalError(err)
		}
		result = append(result, map[string]string{"agent_id": id, "name": name, "model_id": model})
	}
	if err := rows.Err(); err != nil {
		return nil, internalError(err)
	}
	return result, nil
}

func decodeParams(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	return json.Unmarshal(raw, target)
}

func invalidParams(message string) *responseError {
	return &responseError{Code: -32602, Message: message}
}

func internalError(err error) *responseError {
	if errors.Is(err, sql.ErrNoRows) {
		return &responseError{Code: -32004, Message: "not found"}
	}
	return &responseError{Code: -32000, Message: "internal error", Data: err.Error()}
}

func write(w http.ResponseWriter, resp response) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return
	}
}
