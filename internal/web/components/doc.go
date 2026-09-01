// Package components renders the read-only web dashboard. Components are
// written in templ (see .agents/skills/templ) and compiled to Go; each
// logical unit of the UI — the page, the kanban board, the task drawer, and
// the agent activity with per-tool rendering — is one component. This package
// also owns the pure rendering helpers (markdown, diffs, tool titles) so the
// HTTP layer stays about request handling, not markup.
package components
