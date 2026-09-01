# `web`

A read-only, realtime kanban dashboard for the task backlog and the agent bus,
served by the application HTTP server (`cmd/main`). It is built with
[Datastar](https://data-star.dev) for reactivity and
[templ](https://templ.guide) for markup: the server renders HTML components
and pushes them into the DOM over SSE as bus events arrive, with no bespoke
client JavaScript.

## What it provides

- `GET /` — the page: a kanban board with one column per task status
  (Created → Started → Completed → Reviewed), Datastar loaded from a CDN, a
  `selected` signal, and a detail drawer. A `data-init="@get('/events')"` opens
  the live stream on load.
- `GET /events` — the live Datastar SSE stream. It subscribes to the bus
  (`agent.>`, `scheduler.>`) and re-patches the board on every task-relevant
  event (pipeline phases, finished/failed), plus on a keepalive tick so
  CLI-driven changes (add/rm) show up too. It also records each task's current
  pipeline phase, which the detail drawer renders.
- `GET /api/tasks` — the `#task-list` kanban fragment, patched by id.
- `GET /api/tasks/{id}` — the `#detail` drawer for one task: spec, status,
  phase, and a rendered transcript of its execution and review sessions.

## Rendering

Markup lives in `internal/web/components` as templ components (see
`.agents/skills/templ`). Each logical unit is a component: `Page`, `Board`,
`TaskCard`, `TaskDrawer`, `Activity`, and per-tool renderers. Run
`go generate ./internal/web/components` after editing a `.templ` file.

The activity transcript renders each tool call with a display title and a
per-tool component instead of raw JSON: `edit` shows the file path and a
syntax-highlighted unified diff, `read`/`glob`/`grep` show their path or
pattern, `go_test`/`go_build` show their output. Prose (task spec, assistant
text, tool output) is rendered as markdown with syntax-highlighted fenced code
via goldmark + chroma.

## How it behaves

The dashboard is **read-only by design**: no run buttons, no command endpoints.
Controls live in the `taskman` CLI, which sends run commands over the bus
(`taskman.command.run`); the server-side runner in `internal/runner` picks them
up and executes the pipeline. Diffs are viewed in the editor, not the dashboard.

## Notes

- The `/events` SSE stream is long-lived; `internal/process/start.go` exempts it
  from the per-request timeout middleware and sets no `WriteTimeout`.
- Datastar's Go SDK (`github.com/starfederation/datastar-go/datastar`) emits
  the patch events; `PatchElementTempl` renders a templ component into a patch.
  See `.agents/skills/datastar` for Datastar usage.