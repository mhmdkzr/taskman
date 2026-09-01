# `process`

System startup — initializes all runtime dependencies, wires them into the shared `App` struct, starts the HTTP server, and boots the agent runtime.

This is the entry point called by `cmd/main/main.go`. It is the only place where the entire system is assembled.

Startup fails fast if any required dependency is unavailable or not ready.

## Startup Sequence

```
 1. Load environment config           (app.Config from env vars)
 2. Connect NATS + JetStream          (streams.CreateStreams)
 3. Register HTTP routes              (internal/register → per-module registries)
 4. Start HTTP server                 (goroutine, with middleware chain)
 5. Start agent runtime               (internal/server on the same NATS connection)
```

The agent runtime (`startAgent`) opens/migrates the SQLite store, builds the scheduler + consumer + request subscription, and drains in-flight runs on shutdown.

## Graceful Shutdown

Listens for `SIGINT` / `SIGTERM`. On signal:
1. Stops the HTTP server
2. Waits for the agent runtime to drain in-flight runs and sub-agents
3. Closes NATS connections