# `web`

A read-only, realtime dashboard for the task backlog and the agent bus, served
by the application HTTP server (`cmd/main`).

## What it provides

- `GET /` — the dashboard: a dark, minimal single page listing tasks with
  status, and a detail pane showing live activity and the task's unified diff.
- `GET /api/overview` — initial snapshot: all tasks plus recent sessions,
  read straight from the SQLite store (read-only handle).
- `GET /events` — Server-Sent Events stream of every live bus event (`agent.>`
  and `scheduler.>`). Each frame is a JSON envelope
  `{"subject": "...", "payload": {...}}`. Subscriptions are transient core
  NATS subscriptions created per connection and torn down on disconnect.

## How it behaves

The dashboard is **read-only by design**: there is no control surface here —
no run buttons, no command endpoints. Everything it renders comes from the
store or the bus. Controls live in the `taskman` CLI, which sends run commands
over the bus (`taskman.command.run`); the server-side runner in
`internal/runner` picks them up and executes the pipeline. The dashboard then
shows that work live.

The diff pane is fed by the `agent.pipeline.diff` event the pipeline publishes
at each review round, so it shows the live, unified diff of the work.

## How it's invoked

The dashboard is registered on the application HTTP server, so it comes up
with `cmd/main` — no separate command needed. Configure the listen address
with `SERVER_BIND_ADDR` and the bus with `NATS_URL`.

```sh
SERVER_BIND_ADDR=127.0.0.1:8080 go run ./cmd/main
# open http://127.0.0.1:8080
```
