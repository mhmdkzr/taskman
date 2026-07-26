# `process`

System startup — initializes all runtime dependencies, wires them into the shared `App` struct, and starts the HTTP server and Temporal worker.

This is the entry point called by `cmd/main/main.go`. It is the only place where the entire system is assembled.

Startup fails fast if any required dependency is unavailable or not ready.

## Startup Sequence

```
 1. Load environment config           (app.Config from env vars)
 2. Open PostgreSQL connection        (pkg/pg)
 3. Run database migrations           (pkg/migrate)
 4. Connect NATS + JetStream          (app.CreateStreams → per-module stream specs)
 5. Connect Temporal                  (workflow + activity worker)
 6. Register HTTP routes              (internal/register → per-module registries)
 7. Register Temporal workers         (internal/register, registers workflows + activities)
 8. Start HTTP server                 (goroutine, with middleware chain)
 9. Start Temporal worker             (begins processing tasks)
10. Start audit log consumer          (API_AUDIT JetStream stream)
```

## Graceful Shutdown

Listens for `SIGINT` / `SIGTERM`. On signal:
1. Stops the HTTP server
2. Drains the Temporal worker
3. Closes NATS and DB connections
