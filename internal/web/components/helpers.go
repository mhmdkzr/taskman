package components

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"github.com/mhmdkzr/taskman/internal/task"
)

// markdownRenderer converts a task's description and specification plan
// (GitHub-flavored markdown - lists, tables, code fences) to HTML. Raw HTML
// in the source is left escaped rather than rendered (no html.WithUnsafe()):
// these strings come from wherever a task was created, not necessarily a
// trusted human, so treat their markdown as content, not as a way to inject
// markup.
var markdownRenderer = goldmark.New(goldmark.WithExtensions(extension.GFM))

// renderMarkdown renders src as sanitized HTML for direct embedding via
// templ.Raw. A source that fails to parse (which goldmark only does on an
// I/O error from the in-memory buffer, never on malformed markdown) falls
// back to the escaped raw text so the section still shows something.
func renderMarkdown(src string) templ.Component {
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(src), &buf); err != nil {
		return templ.Raw(src)
	}
	return templ.Raw(buf.String())
}

// statusClass maps a task state to the CSS class its status pill uses.
// Active work is amber, a good outcome is green, a terminal non-success or a
// task needing intervention is red, and an unstarted task is neutral.
func statusClass(state task.TaskState) string {
	switch state {
	case task.StateCompleted:
		return "completed"
	case task.StateAbandoned, task.StateBlocked:
		return "interrupted"
	case task.StateSpecify:
		return "neutral"
	default:
		return "running"
	}
}

// statusLabel renders a task state for display.
func statusLabel(state task.TaskState) string {
	switch state {
	case task.StateSpecify:
		return "Specify"
	case task.StateSpecificationReview:
		return "Spec Review"
	case task.StateImplement:
		return "Implement"
	case task.StateVerify:
		return "Verify"
	case task.StateFixVerificationFailure:
		return "Fix Verify"
	case task.StateFixAutomatedReviewFindings:
		return "Fix Review"
	case task.StateAutomatedReview:
		return "Automated Review"
	case task.StateCommit:
		return "Commit"
	case task.StateHumanReview:
		return "Human Review"
	case task.StateFixHumanReviewFindings:
		return "Fix Human Review"
	case task.StateMerge:
		return "Merge"
	case task.StateBlocked:
		return "Blocked"
	case task.StateCompleted:
		return "Completed"
	case task.StateAbandoned:
		return "Abandoned"
	default:
		return state.String()
	}
}

// taskTitle is a task's one-line heading: its title, or - since the title may
// be empty where the description is required - a truncated description.
func taskTitle(t task.Task) string {
	if t.Definition.Title != "" {
		return t.Definition.Title
	}
	return truncate(t.Definition.Description, 120)
}

// truncate collapses whitespace and shortens s to at most n bytes, marking a
// cut with an ellipsis.
func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// label is one task label, split out so labels can be rendered in a stable
// order.
type label struct {
	Key   string
	Value string
}

// sortedLabels returns a task's labels sorted by key. A map's iteration order
// is randomized, so rendering labels directly would reshuffle them on every
// render - and, since the SSE stream patches only when the rendered markup
// changes, would make the page patch itself on every poll for no real change.
func sortedLabels(labels map[string]string) []label {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := make([]label, 0, len(keys))
	for _, key := range keys {
		out = append(out, label{Key: key, Value: labels[key]})
	}
	return out
}

// taskUpdatedAt is when a task last changed: the timestamp of its most recent
// state transition.
func taskUpdatedAt(t task.Task) time.Time {
	if len(t.StateHistory) == 0 {
		return time.Time{}
	}
	return t.StateHistory[len(t.StateHistory)-1].At
}

// taskMeta is the one-line summary shown on a collapsed card: how many state
// transitions the task has, and when it last changed.
func taskMeta(t task.Task) string {
	meta := intString(len(t.StateHistory)) + " transitions"
	if updated := formatTime(taskUpdatedAt(t)); updated != "" {
		meta += " · updated " + updated
	}
	return meta
}

// reviewSummary renders a review configuration as "none", or the required
// reviewers ("agent", "human", or both).
func reviewSummary(cfg task.ReviewConfiguration) string {
	var required []string
	if cfg.Agent.Required {
		required = append(required, "agent")
	}
	if cfg.Human.Required {
		required = append(required, "human")
	}
	if len(required) == 0 {
		return "none"
	}
	return strings.Join(required, " + ")
}

// verificationSummary renders a verification configuration as "none", or the
// configured checks.
func verificationSummary(v task.Verification) string {
	var checks []string
	if v.Tests.Unit {
		checks = append(checks, "unit")
	}
	if v.Tests.Integration {
		checks = append(checks, "integration")
	}
	if v.Tests.EndToEnd {
		checks = append(checks, "end-to-end")
	}
	if v.Linters {
		checks = append(checks, "linters")
	}
	if len(checks) == 0 {
		return "none"
	}
	return strings.Join(checks, ", ")
}

// gitHasFacts reports whether an implementation has any Git facts worth
// rendering, so the Git section can be omitted entirely when it doesn't.
func gitHasFacts(g task.Git) bool {
	return g.Branch != "" || g.Worktree != "" || len(g.Commits) > 0 || g.Merge != nil
}

// formatTime renders a timestamp for display in local time; the zero time
// renders as an empty string rather than a bogus date.
func formatTime(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.Format("2006-01-02 15:04")
}

// checkClass maps a verification check result to its pill class.
func checkClass(result task.CheckResult) string {
	if result == task.CheckOK {
		return "completed"
	}
	return "interrupted"
}

// taskSection is one labeled group of tasks sharing the same workflow bucket
// (see groupTasks), in the order the task list renders them. Key is the
// bucket's stable, JS-safe identifier, used to key its collapse signal.
type taskSection struct {
	Key   string
	Label string
	Tasks []task.Task
}

// groupTasks buckets tasks by what they need next into the order a task list
// should read in: what needs intervention first, then what's paused waiting on
// a review, then active work, then what's finished. Tasks within a bucket keep
// the order they arrived in (oldest first, per store.List).
func groupTasks(tasks []task.Task) []taskSection {
	order := []struct {
		key, label string
	}{
		{"attention", "Needs attention"},
		{"waiting", "Waiting on review"},
		{"active", "In progress"},
		{"done", "Done"},
	}
	buckets := map[string][]task.Task{}
	for _, t := range tasks {
		buckets[bucketFor(t.State())] = append(buckets[bucketFor(t.State())], t)
	}

	sections := make([]taskSection, 0, len(order))
	for _, o := range order {
		if len(buckets[o.key]) == 0 {
			continue
		}
		sections = append(sections, taskSection{Key: o.key, Label: o.label, Tasks: buckets[o.key]})
	}
	return sections
}

// bucketFor maps a state to the groupTasks bucket it belongs to.
func bucketFor(state task.TaskState) string {
	switch state {
	case task.StateBlocked:
		return "attention"
	case task.StateSpecificationReview, task.StateHumanReview:
		return "waiting"
	case task.StateCompleted, task.StateAbandoned:
		return "done"
	default:
		return "active"
	}
}

// jsBool renders a bool as a JS literal, for building Datastar attribute
// expressions (e.g. data-signals:x) where the value must parse as JS.
func jsBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// openSignal names the Datastar signal tracking whether one collapsible card
// is expanded, given some ID unique to it. IDs here are UUIDs, which aren't
// valid JS identifiers as-is (hyphens), hence the substitution.
func openSignal(prefix, id string) string {
	return prefix + "_" + strings.ReplaceAll(id, "-", "_")
}

// openAttrs declares a panel's open/closed signal the first time it's rendered,
// via the __ifmissing modifier - critical here, since the task list patches
// itself on an ambient poll and a plain (re-)declaration would otherwise reset
// every expanded panel back to collapsed on the next tick.
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

func taskOpenAttrs(taskID string) templ.Attributes { return openAttrs("task_open", taskID, false) }
func taskOpenExpr(taskID string) string            { return openExpr("task_open", taskID) }
func taskToggleAction(taskID string) string        { return toggleAction("task_open", taskID) }

// Sections default open: collapsing is a convenience, not the initial state.
func sectionOpenAttrs(key string) templ.Attributes { return openAttrs("section_open", key, true) }
func sectionOpenExpr(key string) string            { return openExpr("section_open", key) }
func sectionToggleAction(key string) string        { return toggleAction("section_open", key) }

// intString renders an int without importing strconv at every call site.
func intString(n int) string {
	return strconv.Itoa(n)
}
