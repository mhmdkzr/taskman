# `web`

A read-only, realtime dashboard for the task backlog and the agent bus, served
by the `taskman web` CLI command.

## What it provides

- `GET /` — the dashboard: a dark, minimal single page listing tasks with
  status, a detail pane showing live activity and the task's unified diff, and
  per-task run controls.
- `GET /api/overview` — initial snapshot: all tasks plus recent sessions,
  read straight from the SQLite store (read-only handle).
- `GET /events` — Server-Sent Events stream of every live bus event (`agent.>`,
  `scheduler.>` and the command subject). Each frame is a JSON envelope
  `{"subject": "...", "payload": {...}}`. Subscriptions are transient core
  NATS subscriptions created per connection and torn down on disconnect.
- `POST /api/commands` — the browser's one write: publish a `RunCommand`
  (`taskman.command.run`) asking the runner to execute the given task ids.
- command consumer — `Run` subscribes to `taskman.command.run` and executes
  each requested task through `pipeline.RunTask` in its own goroutine and
  worktree, so concurrent commands run concurrently.

## How it behaves

The UI is read-only by design: everything it renders comes from the store or
the bus. The only action it can trigger is running tasks, and that goes
through the command subject — a request, not a direct write — so a different
runner could consume the same commands.

The diff pane is fed by the `agent.pipeline.diff` event the pipeline publishes
at each review round, so it shows the live, unified diff of the work.

## How it's invoked

```sh
taskman web -addr 127.0.0.1:8080 -source /path/to/repo
# open http://127.0.0.1:8080
```

Requires the same `AGENT_*` provider configuration as `taskman run`, plus
NATS with JetStream (for the bus). Blocks until interrupted.