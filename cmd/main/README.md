# `cmd/main`

Application entry point. Starts the application by calling `process.Start(ctx)`. Handles OS signals (SIGINT, SIGTERM) for graceful shutdown.

Usage:

```sh
go run ./cmd/main
```

All configuration is read from environment variables (see `.env.example`).
