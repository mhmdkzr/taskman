# `process`

System startup — initializes all runtime dependencies, wires them into the shared `App` struct, and starts the HTTP server and Temporal worker.

This is the entry point called by `cmd/main/main.go`. It is the only place where the entire system is assembled.

Startup fails fast if any required dependency is unavailable or not ready.

## Startup Sequence

```
 1. Load environment config           (app.Config from env vars)
2. Open SQLite connection            (app config)
3. Run database migrations           (pkg/migrate)
4. Connect Temporal                  (workflow + activity worker)
 5. Register HTTP routes              (internal/app/register → per-module registries)
 6. Register Temporal workers         (internal/app/register, registers workflows + activities)
 7. Start HTTP server                 (goroutine, with middleware chain)
 8. Start Temporal worker             (begins processing tasks)
```

## Graceful Shutdown

Listens for `SIGINT` / `SIGTERM`. On signal:
1. Stops the HTTP server
2. Drains the Temporal worker
3. Closes the DB connection
