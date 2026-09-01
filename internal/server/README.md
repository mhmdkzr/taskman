# `server`

The long-running agent runtime. It owns the store, the shared tool set, the durable scheduler and the scheduled-run consumer, and answers client commands arriving as NATS core request/reply on `protocol.Subject` (`taskman.request`).

## What it provides

- `New` — assembles the runtime: opens/migrates the SQLite store, builds the scheduler, tools and consumer.
- `Run` — serves until the context is cancelled: starts the scheduler and consumer, subscribes to the request subject, and dispatches `run` / `ping` requests one at a time.
- `Close` — releases the store.

## How it's invoked

`internal/process/start.go` calls `server.New(opts, pub)` and runs it on the same NATS connection as the HTTP server.

Requests are handled synchronously in the subscription callback, so NATS serializes them; a `run` reply blocks until the run completes.