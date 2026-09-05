# loop

Go application starter using SQLite, structured logging,
HTTP middleware, and vertical slices.

## Quick Start

```bash
cp .env.example .env
go run ./cmd/main
```

The server reads configuration from `.env` and listens on `SERVER_BIND_ADDR`.
The application stores its data in SQLite.

## Project Layout

| Path | Purpose |
| --- | --- |
| `cmd/` | Server and migration entry points |
| `internal/` | Domain logic grouped by module and slice |
| `pkg/` | Shared libraries |
| `migrations/` | Database schema migrations |
| `config/` | Application configuration |

## Commands

```bash
make lint
make fmt
make test
make test-db
```

## Dependencies

- Go 1.27+
- SQLite

## Environment

`.env.example` contains the current application configuration. It covers the
HTTP server, logger, SQLite, and model provider,
Telegram, Tavily, and browser tools.

## Health

- `GET /health` provides the liveness response.
