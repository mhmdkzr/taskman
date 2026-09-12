# `internal/task/view/md`

Renders a task as a human-readable Markdown document (the CLI's `--md`), for display outside
taskman itself - a PR description or a chat message - not as a replacement for `--json`, which
stays the machine-readable source of truth.

`Render(ctx, st, id)` reads the task from the store; `RenderTask(t)` renders one already loaded.
Both execute the embedded `template.md` (see `embed.go`). Sections a task does not yet have
(Specification, Implementation, Blocked, Abandoned, a Git merge, ...) are omitted rather than shown
empty.
