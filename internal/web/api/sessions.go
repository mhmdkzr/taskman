package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"uuid"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/mhmdkzr/loop/internal/agent"
	"github.com/mhmdkzr/loop/internal/agent/sessions"
	"github.com/mhmdkzr/loop/internal/agent/tools/ask"
	"github.com/mhmdkzr/loop/internal/agent/tools/task"
	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/web/components"
	"github.com/mhmdkzr/loop/pkg/jsonresp"
)

// createSessionRequest is the body of POST /sessions: the agent to run as and
// the first message to send it.
type createSessionRequest struct {
	AgentName string `json:"agent_name"`
	Prompt    string `json:"prompt"`
}

// respondRequest is the body of POST /sessions/{id}/messages: the next
// message to send an existing session.
type respondRequest struct {
	Prompt string `json:"prompt"`
}

// answerAskRequest is the body of POST /sessions/{id}/asks/{ask_id}/answer:
// a human's answer to a pending ask.PendingAsk. Datastar posts every signal
// on the page, not just these two (see components.pendingAskView), so the
// extra fields are simply ignored by json.Decode.
type answerAskRequest struct {
	Selected []int  `json:"ask_selected"`
	Custom   string `json:"ask_custom"`
}

// indexHandler renders the app shell with the most recently active session
// selected, or the new-session composer if no session exists yet.
func indexHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summaries, err := sessions.ListSessions(r.Context(), a.Deps.Store)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), fmt.Errorf("list sessions: %w", err))
			return
		}
		var active *sessions.SessionID
		if len(summaries) > 0 {
			active = &summaries[0].SessionID
		}
		renderPage(w, r, a, active)
	}
}

// newSessionPageHandler renders the app shell with the new-session composer
// active, regardless of whether other sessions already exist.
func newSessionPageHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, r, a, nil)
	}
}

// sessionPageHandler renders the app shell with one session active, for
// direct navigation (a bookmarked or shared session URL).
func sessionPageHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseSessionID(r.PathValue("id"))
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}
		renderPage(w, r, a, &id)
	}
}

// createSessionHandler starts a new session for the chosen agent, runs its
// first turn synchronously, and patches the app shell to make it active. It
// is a Datastar action (see the "+ New session" composer), not a page load,
// so on success it patches HTML over SSE rather than navigating.
func createSessionHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := createSessionRequestFromHTTP(r)
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}

		id, err := agent.StartSession(r.Context(), a.Deps.Store, a.Cfg.Provider, req.AgentName, nil)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), fmt.Errorf("start session: %w", err))
			return
		}

		if _, err := agent.Respond(r.Context(), a.Deps.AgentTools, id, req.Prompt); err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), fmt.Errorf("respond: %w", err))
			return
		}

		patchPage(w, r, a, &id)
	}
}

// respondHandler runs one turn of an existing session and patches the app
// shell with the updated transcript. It is a Datastar action (see the
// composer), not a page load.
func respondHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseSessionID(r.PathValue("id"))
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}
		req, err := respondRequestFromHTTP(r)
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if _, err := agent.Respond(r.Context(), a.Deps.AgentTools, id, req.Prompt); err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), fmt.Errorf("respond: %w", err))
			return
		}

		patchPage(w, r, a, &id)
	}
}

// answerAskHandler records a human's answer to a pending ask. It does not
// itself run the turn to completion - the ask tool call blocked on this
// answer is polling for it (see ask.execute) from within the original
// POST .../messages request, which is what actually resumes generation and
// eventually patches the page with the finished turn. This handler just
// persists the answer and patches the current (still "running") state, so
// the ask card disappears immediately; the page's own interval poll (see
// App) picks up the turn's eventual completion.
func answerAskHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, err := parseSessionID(r.PathValue("id"))
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}
		askID, err := parseAskID(r.PathValue("ask_id"))
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}
		req, err := answerAskRequestFromHTTP(r)
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if err := ask.AnswerAsk(r.Context(), a.Deps.Store.RW(), askID, req.Selected, req.Custom); err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), fmt.Errorf("answer ask: %w", err))
			return
		}

		view, err := buildAppView(r.Context(), a, &sessionID)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
			return
		}
		sse := datastar.NewSSE(w, r)
		if err := sse.PatchElementTempl(components.App(view)); err != nil {
			slog.Error("patch app view", "error", err)
		}
		// The turn isn't necessarily done just because this ask was answered
		// - it may still be generating, or hit another ask - so keep polling
		// (see refreshSessionHandler, which reports the real answer once the
		// blocked .../messages request actually completes).
		if err := sse.MarshalAndPatchSignals(map[string]any{
			"ask_selected": []int{}, "ask_custom": "", "turn_running": true,
		}); err != nil {
			slog.Error("reset ask signals", "error", err)
		}
	}
}

// refreshSessionHandler patches sessionID's session view without disturbing
// anything else the page has going on. It's what the page's interval poll
// hits (see components.pollAction) while a turn is running, so a pending ask
// - or the turn simply finishing - shows up without the user refreshing. It
// deliberately does not touch the composer's "prompt" signal, unlike
// patchPage: a passive background refresh must not blow away a message the
// user is mid-typing.
func refreshSessionHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseSessionID(r.PathValue("id"))
		if err != nil {
			jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
			return
		}
		view, err := buildAppView(r.Context(), a, &id)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
			return
		}
		sse := datastar.NewSSE(w, r)
		if err := sse.PatchElementTempl(components.App(view)); err != nil {
			slog.Error("patch app view", "error", err)
		}
		running := false
		if view.Active != nil && len(view.Active.Turns) > 0 {
			running = view.Active.Turns[len(view.Active.Turns)-1].Status == "running"
		}
		if err := sse.MarshalAndPatchSignals(map[string]any{"turn_running": running}); err != nil {
			slog.Error("update turn_running signal", "error", err)
		}
	}
}

func createSessionRequestFromHTTP(r *http.Request) (createSessionRequest, error) {
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return createSessionRequest{}, fmt.Errorf("decode request: %w", err)
	}
	if strings.TrimSpace(req.AgentName) == "" {
		return createSessionRequest{}, fmt.Errorf("agent_name is required")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return createSessionRequest{}, fmt.Errorf("prompt is required")
	}
	return req, nil
}

func respondRequestFromHTTP(r *http.Request) (respondRequest, error) {
	var req respondRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return respondRequest{}, fmt.Errorf("decode request: %w", err)
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return respondRequest{}, fmt.Errorf("prompt is required")
	}
	return req, nil
}

func answerAskRequestFromHTTP(r *http.Request) (answerAskRequest, error) {
	var req answerAskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return answerAskRequest{}, fmt.Errorf("decode request: %w", err)
	}
	return req, nil
}

func parseSessionID(raw string) (sessions.SessionID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return sessions.SessionID{}, fmt.Errorf("parse session id: %w", err)
	}
	return sessions.SessionID(id), nil
}

func parseAskID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parse ask id: %w", err)
	}
	return id, nil
}

// httpStatusForError maps a domain error to its HTTP status; anything
// unrecognized is a 500, per CLAUDE.md's error-handling conventions.
func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, sessions.ErrSessionNotFound), errors.Is(err, sessions.ErrAgentNotFound),
		errors.Is(err, task.ErrTaskNotFound), errors.Is(err, ask.ErrAskNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// renderPage renders the full app shell for a normal page load (GET): direct
// navigation, a bookmarked URL, or a browser refresh.
func renderPage(w http.ResponseWriter, r *http.Request, a app.App, activeID *sessions.SessionID) {
	view, err := buildAppView(r.Context(), a, activeID)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	if err := components.App(view).Render(r.Context(), w); err != nil {
		slog.Error("render app page", "error", err)
	}
}

// patchPage renders the full app shell and pushes it as a Datastar SSE patch:
// the response to a create-session or send-message action, which the browser
// morphs into place by element id rather than navigating. It also resets the
// composer's "prompt" signal - the morphed textarea's own emptiness doesn't
// clear it, since Datastar's bound signal, not the incoming markup, decides
// what the element displays.
func patchPage(w http.ResponseWriter, r *http.Request, a app.App, activeID *sessions.SessionID) {
	view, err := buildAppView(r.Context(), a, activeID)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	sse := datastar.NewSSE(w, r)
	if err := sse.PatchElementTempl(components.App(view)); err != nil {
		slog.Error("patch app view", "error", err)
	}
	// The action that reaches patchPage (create session, send message) only
	// returns once its agent.Respond call has - so whatever turn it just ran
	// is no longer running by construction, and the interval poll started by
	// sendAction/createAction can stop.
	if err := sse.MarshalAndPatchSignals(map[string]any{"prompt": "", "turn_running": false}); err != nil {
		slog.Error("reset composer signal", "error", err)
	}
}

// buildAppView loads everything components.App needs to render: every
// session for the rail, and - when activeID is non-nil - that session's full
// detail for the main pane.
func buildAppView(ctx context.Context, a app.App, activeID *sessions.SessionID) (components.AppView, error) {
	summaries, err := sessions.ListSessions(ctx, a.Deps.Store)
	if err != nil {
		return components.AppView{}, fmt.Errorf("list sessions: %w", err)
	}
	agentNames, err := agent.ListAgentNames(ctx, a.Deps.Store)
	if err != nil {
		return components.AppView{}, fmt.Errorf("list agent names: %w", err)
	}

	view := components.AppView{Mode: "sessions", Sessions: summaries, AgentNames: agentNames}
	if activeID == nil {
		return view, nil
	}
	detail, err := sessions.GetSessionDetail(ctx, a.Deps.Store, *activeID)
	if err != nil {
		return components.AppView{}, fmt.Errorf("get session detail: %w", err)
	}

	// A pending ask can only exist on a still-running turn; skip the lookup
	// for a session that has none (the common case) rather than querying
	// session_asks once per completed turn.
	pendingAsks := make(map[string]*ask.PendingAsk)
	for _, t := range detail.Turns {
		if t.Status != "running" {
			continue
		}
		pending, err := ask.PendingAskForTurn(ctx, a.Deps.Store.RO(), t.TurnID)
		if err != nil {
			return components.AppView{}, fmt.Errorf("get pending ask: %w", err)
		}
		if pending != nil {
			pendingAsks[t.TurnID] = pending
		}
	}

	view.Active = &components.SessionView{SessionDetail: detail, PendingAsks: pendingAsks}
	return view, nil
}
