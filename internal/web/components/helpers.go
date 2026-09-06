package components

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
)

// formatTime renders an RFC3339Nano timestamp as a local HH:MM clock time for
// display; a value that fails to parse is shown verbatim rather than hidden.
func formatTime(iso string) string {
	t, err := time.Parse(time.RFC3339Nano, iso)
	if err != nil {
		return iso
	}
	return t.Local().Format("15:04") //nolint:gosmopolitan // display in the operator's own local time, by design
}

// formatTokens renders a token count with thousands separators.
func formatTokens(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// prettyJSON indents raw for display; a value that isn't valid JSON (a plain
// string tool output, say) is shown verbatim rather than hidden.
func prettyJSON(raw []byte) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
}

// truncate collapses whitespace and shortens s to at most n runes, marking a
// cut with an ellipsis.
func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// statusClass maps a turn/session/task status to the CSS class its dot/pill
// uses. Turn statuses (running/completed/interrupted) and task states
// (created/started/completed/cancelled/blocked/failed) share one vocabulary:
// active work is amber, a good outcome is green, something needing attention
// is red, and anything not yet or no longer active is neutral.
func statusClass(status string) string {
	switch status {
	case "running", "started":
		return "running"
	case "completed":
		return "completed"
	case "interrupted", "blocked", "failed":
		return "interrupted"
	default: // "new", "created", "cancelled"
		return "neutral"
	}
}

// statusLabel renders a status for display.
func statusLabel(status string) string {
	switch status {
	case "running":
		return "Running"
	case "started":
		return "Started"
	case "completed":
		return "Completed"
	case "interrupted":
		return "Interrupted"
	case "blocked":
		return "Blocked"
	case "failed":
		return "Failed"
	case "created":
		return "Created"
	case "cancelled":
		return "Cancelled"
	case "new":
		return "New"
	default:
		return status
	}
}

// meterStyle renders a 1-5 planning level as an inline width style for its
// meter fill.
func meterStyle(level int) templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf("width:%d%%", level*100/5))
}

// itoa renders an int for display.
func itoa(n int) string {
	return strconv.Itoa(n)
}

// jsBool renders a bool as a JS literal, for building Datastar attribute
// expressions (e.g. data-signals:x) where the value must parse as JS.
func jsBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// jsString quotes s as a JS string literal, for building Datastar attribute
// expressions (e.g. data-signals:x) where the value must parse as JS.
func jsString(s string) string {
	return strconv.Quote(s)
}

// sendAction is the Datastar action for submitting a message into an existing
// session: it posts the bound "prompt" signal. The server clears it in the
// patch response (see api.patchPage) rather than the client clearing it
// itself, since the reply only arrives once the turn finishes. It also flips
// $turn_running immediately (client-side, before the server has responded at
// all) so the page's own interval poll (see pollAction) starts checking for
// progress - most importantly a pending ask - right away, rather than only
// after the very request it's polling about finally completes.
func sendAction(sessionID string) string {
	return fmt.Sprintf("$turn_running = true; @post('/sessions/%s/messages')", sessionID)
}

// sendKeydownAction submits on Enter (without Shift, which inserts a newline).
func sendKeydownAction(sessionID string) string {
	return fmt.Sprintf(
		"evt.key === 'Enter' && !evt.shiftKey && (evt.preventDefault(), %s)",
		sendAction(sessionID),
	)
}

// createAction is the Datastar action for starting a new session: it posts
// the bound "prompt" and "agent_name" signals.
const createAction = "@post('/sessions')"

// createKeydownAction submits on Enter (without Shift).
const createKeydownAction = "evt.key === 'Enter' && !evt.shiftKey && (evt.preventDefault(), " + createAction + ")"

// refreshAction is the Datastar action for the page's ambient background
// poll (see the data-on-interval on #main-pane in App): it keeps the rail and
// the active item in sync with changes made elsewhere - most importantly an
// MCP client creating or driving a session while this page is open - without
// the user refreshing. It always fires, not just while a turn is running,
// since a new session or task can appear at any time. Each branch hits a
// dedicated refresh endpoint rather than the page route itself, so the
// response is an SSE patch like every other action, not a full HTML document.
func refreshAction(view AppView) string {
	switch {
	case view.Mode == "tasks" && view.ActiveTask != nil:
		return fmt.Sprintf("@get('/tasks/%s/refresh')", view.ActiveTask.Task.ID.String())
	case view.Mode == "tasks":
		return "@get('/tasks/refresh')"
	case view.Active != nil:
		return fmt.Sprintf("@get('/sessions/%s/refresh')", view.Active.SessionID.String())
	default:
		return "@get('/sessions/refresh')"
	}
}

// askOptionSelectedExpr reads whether option id is currently selected.
func askOptionSelectedExpr(id int) string {
	return fmt.Sprintf("$ask_selected.includes(%d)", id)
}

// askOptionClickAction toggles option id into $ask_selected for a
// multi-select ask, or replaces the selection for a single-select one.
func askOptionClickAction(id int, multiSelect bool) string {
	if multiSelect {
		return fmt.Sprintf(
			"$ask_selected = $ask_selected.includes(%d) ? $ask_selected.filter(x => x !== %d) : [...$ask_selected, %d]",
			id, id, id,
		)
	}
	return fmt.Sprintf("$ask_selected = [%d]", id)
}

// answerAskAction posts the bound $ask_selected/$ask_custom signals as the
// answer to askID.
func answerAskAction(sessionID, askID string) string {
	return fmt.Sprintf("@post('/sessions/%s/asks/%s/answer')", sessionID, askID)
}
