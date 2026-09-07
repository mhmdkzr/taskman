package components

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"github.com/mhmdkzr/loop/internal/agent/tools/task"
)

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

// prettyJSON indents raw for display; a value that isn't valid JSON is shown as is.
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

// statusClass maps a turn/session/task status to the CSS class its pill uses.
// Turn statuses (running/completed/interrupted) and task states
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

// levelLabel renders a 1-5 planning level (see task.Level) as the same word
// its own jsonschema description already uses ("very-low" .. "very-high").
func levelLabel(level int) string {
	switch level {
	case 1:
		return "Very Low"
	case 2:
		return "Low"
	case 3:
		return "Medium"
	case 4:
		return "High"
	case 5:
		return "Very High"
	default:
		return strconv.Itoa(level)
	}
}

// taskTotalTokens sums the token usage of every session dispatched for a
// task, across its whole turn history.
func taskTotalTokens(t TaskDetailView) int64 {
	var total int64
	for _, row := range t.Sessions {
		total += row.Session.TotalTokens
	}
	return total
}

// modelProvider guesses a model's provider from its name for display, since
// task.Task only stores the model string itself. Unrecognized names return
// "" so the cell is simply omitted rather than showing a wrong guess.
func modelProvider(model string) string { // TODO: complete, and find a more reliable way (db?)
	switch {
	case strings.HasPrefix(model, "claude"):
		return "Anthropic"
	case strings.HasPrefix(model, "gpt"), strings.HasPrefix(model, "o1"),
		strings.HasPrefix(model, "o3"), strings.HasPrefix(model, "o4"):
		return "OpenAI"
	case strings.HasPrefix(model, "gemini"):
		return "Google"
	default:
		return ""
	}
}

// taskSection is one labeled group of tasks sharing the same urgency bucket
// (see groupTasksByState), in the order the task list renders them.
type taskSection struct {
	Label string
	Tasks []TaskDetailView
}

// groupTasksByState buckets tasks by state into the order a task list should
// read in: what needs attention first, then what's actively running, then
// what's waiting to start, then what's finished. Tasks within a bucket keep
// the order they arrived in (oldest first, per buildTasksView).
func groupTasksByState(tasks []TaskDetailView) []taskSection {
	buckets := map[string][]TaskDetailView{}
	for _, t := range tasks {
		key := "queued"
		switch t.Task.State {
		case task.TaskStateCreated:
			key = "queued"
		case task.TaskStateStarted:
			key = "active"
		case task.TaskStateBlocked, task.TaskStateFailed:
			key = "attention"
		case task.TaskStateCompleted, task.TaskStateCancelled:
			key = "done"
		}
		buckets[key] = append(buckets[key], t)
	}

	order := []struct{ key, label string }{
		{"attention", "Needs attention"},
		{"active", "Active"},
		{"queued", "Queued"},
		{"done", "Done"},
	}
	sections := make([]taskSection, 0, len(order))
	for _, o := range order {
		if len(buckets[o.key]) == 0 {
			continue
		}
		sections = append(sections, taskSection{Label: o.label, Tasks: buckets[o.key]})
	}
	return sections
}

// jsBool renders a bool as a JS literal, for building Datastar attribute
// expressions (e.g. data-signals:x) where the value must parse as JS.
func jsBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// markdownRenderer converts task specifications (GitHub-flavored markdown -
// lists, tables, code fences) to HTML. Raw HTML in the source is left
// escaped rather than rendered (no html.WithUnsafe()): specs come from
// wherever a task was created, not necessarily a trusted human at a
// keyboard, so treat their markdown as content, not as a way to inject markup.
var markdownRenderer = goldmark.New(goldmark.WithExtensions(extension.GFM))

// renderMarkdown renders src as sanitized HTML for direct embedding via
// @templ.Raw. A source that fails to parse (which goldmark only does on an
// I/O error from the in-memory buffer, never on malformed markdown) falls
// back to the escaped raw text so the section still shows something.
func renderMarkdown(src string) templ.Component {
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(src), &buf); err != nil {
		return templ.Raw(src)
	}
	return templ.Raw(buf.String())
}

// openSignal names the Datastar signal tracking whether one collapsible
// panel (a task card or a session panel) is expanded, given some ID unique
// to it. IDs here are UUIDs, which aren't valid JS identifiers as-is
// (hyphens), hence the substitution.
func openSignal(prefix, id string) string {
	return prefix + "_" + strings.ReplaceAll(id, "-", "_")
}

// openAttrs declares a panel's open/closed signal the first time it's
// rendered, via the __ifmissing modifier - critical here, since the task
// list patches itself on an ambient poll (see the data-on-interval in App)
// and a plain (re-)declaration would otherwise reset every expanded panel
// back to collapsed on the next tick.
func openAttrs(prefix, id string, openByDefault bool) templ.Attributes {
	return templ.Attributes{"data-signals:" + openSignal(prefix, id) + "__ifmissing": jsBool(openByDefault)}
}

// openExpr reads a panel's open/closed signal.
func openExpr(prefix, id string) string {
	return "$" + openSignal(prefix, id)
}

// toggleAction flips a panel's open/closed signal.
func toggleAction(prefix, id string) string {
	expr := openExpr(prefix, id)
	return expr + " = !" + expr
}

func sessionOpenAttrs(sessionID string) templ.Attributes {
	return openAttrs("session_open", sessionID, false)
}
func sessionOpenExpr(sessionID string) string     { return openExpr("session_open", sessionID) }
func sessionToggleAction(sessionID string) string { return toggleAction("session_open", sessionID) }

func taskOpenAttrs(taskID string, openByDefault bool) templ.Attributes {
	return openAttrs("task_open", taskID, openByDefault)
}
func taskOpenExpr(taskID string) string     { return openExpr("task_open", taskID) }
func taskToggleAction(taskID string) string { return toggleAction("task_open", taskID) }
