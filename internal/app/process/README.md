# `process`

System startup — initializes all runtime dependencies, wires them into the shared `App` struct, and starts the HTTP server.

This is the entry point called by `cmd/main/main.go`. It is the only place where the entire system is assembled.

Startup fails fast if any required dependency is unavailable or not ready.

## Startup Sequence

```
 1. Load environment config           (app.Config from env vars)
 2. Open SQLite connection            (app config)
 3. Run database migrations           (pkg/migrate)
 4. Seed agent data                    (providers, model, tools)
 5. Build runtime clients              (telegram, browser, websearch)
 6. Register HTTP routes              (internal/app/register → per-module registries)
 7. Start HTTP server                 (goroutine, with middleware chain)
```

## Graceful Shutdown

Listens for `SIGINT` / `SIGTERM`. On signal:
1. Stops the HTTP server
2. Closes the DB connection
