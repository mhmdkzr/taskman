# `internal/web`

Serves taskman's read-only web UI: one page listing every task, grouped by what it needs next,
with each task expanding in place to its full detail. It is the one long-running frontend
(`taskman --web`), as opposed to the one-shot CLI commands and MCP tool calls.

## Running

```bash
taskman --web --db ./tasks.db --port 8080
```

The page is kept live over datastar/SSE: it opens a stream to `/tasks`, and the server re-renders
the task list on a one-second poll, patching the `#tasks` element only when the rendered markup
changes.

## Layout

- `web.go` - the `Server`: HTTP routes (`GET /` page, `GET /tasks` SSE stream), graceful shutdown
  (`Run`), and the store read via the `list` command.
- `components/` - the server-rendered UI. `Page` is the HTML shell (fonts, dark-only palette, CSS,
  datastar bundle); `TaskList` is the grouped list of expandable task cards.

`components.TasksID` is shared between the page template and the stream handler so the patch
selector can never drift from the rendered markup.

## Rendering

The list groups tasks into collapsible sections - Needs attention, Waiting on review, In progress,
and Done - so a busy category can be folded away; sections start expanded. Each task card shows a
one-line summary (title, state pill, labels, transition count) and expands to its description and
specification plan (rendered from Markdown via goldmark), recorded workflow configuration, Git
facts, blockage/abandonment, verification attempts, review results, and its full state history.

Section and card disclosure state lives in datastar signals declared with `__ifmissing`, not native
`<details open>` or client-only CSS classes: the list patches itself on the ambient poll, and a
plain re-declaration would reset every expanded card on the next tick.

## Tests

`components/components_test.go` renders fully populated and bare tasks and asserts the detail
sections appear, plus the state-grouping and label helpers.
