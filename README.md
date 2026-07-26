# app

Generic Go application starter. Provides a production-ready foundation with PostgreSQL, NATS/JetStream, Temporal, structured logging, HTTP middleware, and vertical slice architecture.

## Architecture

Inbound requests flow through the REST API. Internal logic is organized as
vertical slices under `internal/`, each owning its HTTP handlers, domain types,
persistence, and Temporal workflows/activities.

## Quick Start

```bash
# Copy and edit configuration
cp .env.example .env

# Run database migrations
cd backend && go run ./cmd/migrate

# Start the server
cd backend && go run ./cmd/main
```

## Project Layout

| Path             | Purpose                                         |
|------------------|--------------------------------------------------|
| `backend/`       | Go source code (module root)                     |
| `backend/cmd/`   | Entry points (server, CLI, migrations)           |
| `backend/internal/` | Domain logic grouped by module and slice      |
| `backend/pkg/`   | Shared libraries (DB, NATS, pagination, logging) |
| `backend/migrations/` | Database schema migrations                 |
| `config/`        | Infrastructure configuration (NATS, Temporal)    |

## Available Commands

```bash
make lint       # fmt + vet + staticcheck + govulncheck + test
make test       # run all tests
```

## Dependencies

- Go 1.26+
- PostgreSQL
- NATS (with JetStream)
- Temporal
