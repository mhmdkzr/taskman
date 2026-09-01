# `web`

A read-only, realtime dashboard for the task backlog and the agent bus, served
by the application HTTP server (`cmd/main`). It is built with
[Datastar](https://data-star.dev): the server renders HTML fragments and
pushes them into the DOM over SSE as bus events arrive, with no bespoke client
JavaScript.

## What it provides

- `GET /` — the page shell: a dark, minimal layout with Datastar loaded from a
  CDN, a `selected` signal, and a server-rendered `#task-list`. A
  `data-init="@get('/events')"` opens the live stream on load.
- `GET /events` — the live Datastar SSE stream. It subscribes to the bus
  (`agent.>`, `scheduler.>`) and re-patches `#task-list` on every task-relevant
  event (pipeline phases, diffs, finished/failed), plus on a keepalive tick so
  CLI-driven changes (add/rm) show up too. It also records the transient live
  state — a task's current phase and latest unified diff — that the detail
  fragment renders.
- `GET /api/tasks` — the `#task-list` fragment, `text/html`, morphed in by id.
- `GET /api/tasks/{id}` — the `#detail` fragment for one task: spec, status,
  phase, session activity (from the store's transcripts) and the latest diff.
  It is fetched on task click and re-fetched on a short interval while the task
  stays selected, so a running task's activity and diff stay live.

## How it behaves

The dashboard is **read-only by design**: no run buttons, no command endpoints.
Controls live in the `taskman` CLI, which sends run commands over the bus
(`taskman.command.run`); the server-side runner in `internal/runner` picks them
up and executes the pipeline. The dashboard then shows that work live.

State flows the Datastar way: signals live client-side (`selected`), and every
request/stream update is a server-rendered HTML fragment morphed into the DOM by
element id. There is no hand-written fetch/EventSource/JSON-rendering code.

## Notes

- The `/events` SSE stream is long-lived; `internal/process/start.go` exempts it
  from the per-request timeout middleware and sets no `WriteTimeout`, so it is
  not cut off (a browser would otherwise loop reconnect/live).
- Datastar's Go SDK (`github.com/starfederation/datastar-go/datastar`) emits
  the patch events; the JS bundle is pinned via CDN in `index.html`. See
  `.agents/skills/datastar/SKILL.md` for how to use it here.
