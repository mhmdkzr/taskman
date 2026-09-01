# `server`

The long-running agent runtime. It uses the process-wide store and shared tool set, the durable scheduler and the scheduled-run consumer, and answers client commands arriving as NATS core request/reply on `protocol.Subject` (`taskman.request`).

## What it provides

- `New` — assembles the runtime over the shared store and publisher: builds the scheduler, tools and consumer.
- `Run` — serves until the context is cancelled: starts the scheduler and consumer, subscribes to the request subject, and dispatches `run` / `ping` requests one at a time.
- `Close` — stops the runtime's background work. The store is shared with the process (see `internal/process/start.go`), so `Close` does not close it.

## How it's invoked

`internal/process/start.go` opens the store, then calls `server.New(opts, pub, st)` and runs it on the same NATS connection as the HTTP server.

Requests are handled synchronously in the subscription callback, so NATS serializes them; a `run` reply blocks until the run completes.
