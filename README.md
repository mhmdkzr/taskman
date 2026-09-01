# taskman

Go application. A NATS/JetStream microservice (HTTP + structured logging + vertical slice architecture) that also runs an LLM agent runtime: an agent with tools executes prompts over NATS request/reply, persists sessions to SQLite, and durably schedules recurring runs.

## Architecture

Inbound requests hit the REST API served by the Go application on `:8080`.
Internal logic is organized as vertical slices under `internal/`, each owning
its HTTP handlers, domain types, and NATS/JetStream event wiring. Alongside the
HTTP server, `internal/process/start.go` boots the agent runtime (`internal/agent`,
`internal/server`, `internal/scheduler`, `internal/store`).

## Quick Start

```bash
# Copy and edit configuration (set AGENT_PROVIDER_BASE_URL / AGENT_PROVIDER_API_KEY)
cp .env.example .env

# Start infrastructure (NATS)
docker compose up -d

# Start the server
go run ./cmd/main
```

## Project Layout

| Path             | Purpose                                              |
|------------------|------------------------------------------------------|
| `cmd/`           | Entry points (server)                                |
| `internal/`      | Domain logic grouped by module and slice             |
| `pkg/`           | Shared libraries (NATS, logging, pagination)         |
| `config/`        | Infrastructure configuration (NATS)                  |

## Agent runtime

The agent listens on NATS subject `taskman.request` (core request/reply). Tools: `read`/`edit`/`glob`/`grep` (see `internal/codebase` and `internal/tools/codebase` — scoped to one git worktree per run, not a general-purpose shell), `spawn_subagent`/`subagent_result`, and `telegram_send`/`telegram_read` (when `AGENT_TELEGRAM_*` is set). Sessions persist to SQLite; scheduled runs are durable via `internal/scheduler`.

## Available Commands

```bash
make lint          # vet + staticcheck + golangci-lint + govulncheck
make fmt           # gofmt + goimports + mdjsonfmt
make test          # unit tests (go test ./...)
make test-e2e      # e2e tests (RUN_E2E_TESTS=1)
```

## Dependencies

- Go 1.27+
- NATS 2.14 with JetStream
- `github.com/zendev-sh/goai` (LLM SDK, vendored)

## Stack — Infrastructure Services

Run `docker compose config` to validate `compose.yaml`.

| Service | Image | Ports (host→container) | Purpose |
|---------|-------|------------------------|---------|
| `app` | `golang:1.27.0-alpine` | — (`expose 8080` only) | Go API; healthcheck `GET /health` |
| `nats` | `nats:2.14.2-alpine3.22` | `127.0.0.1:4222:4222` | JetStream |

## Environment

- `.env` — app server (copy from `.env.example`): `SERVER_BIND_ADDR`, `NATS_URL`, `LOGGER_*`, `AGENT_*` (`AGENT_PROVIDER_BASE_URL`, `AGENT_PROVIDER_API_KEY`, `AGENT_DB_PATH`, `AGENT_MODEL`, `AGENT_REASONING_EFFORT`, `AGENT_MAX_STEPS`, `AGENT_TELEGRAM_API_KEY`, `AGENT_TELEGRAM_CHANNEL_ID`)
- `.env.compose` — compose overrides (`NATS_URL=nats://nats:4222`)

Health:

- `GET /health` → `{"status":"ok","commit":"<short-sha>"}` (`pkg/githash`, `internal/health/get`) — liveness, always 200
- `GET /ready` → per-dependency probes (NATS, JetStream) with 5s timeout — `internal/health/ready`

Built Docker image (`Dockerfile`) is `golang:1.27.0-alpine` builder + `alpine:3.24` runtime, `GOOS=linux CGO_ENABLED=1 -mod=vendor`.
