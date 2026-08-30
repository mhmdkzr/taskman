# app

Generic Go application starter. Provides a production-ready foundation with PostgreSQL, NATS/JetStream, Temporal, structured logging, HTTP middleware, and vertical slice architecture.

## Architecture

Inbound requests hit Caddy (`:8090` → `app:8080`) and flow through the REST API.
Internal logic is organized as vertical slices under `internal/`, each owning
its HTTP handlers, domain types, persistence, and Temporal workflows/activities.

## Quick Start

```bash
# Copy and edit configuration
cp .env.example .env

# Start infrastructure (PostgreSQL, NATS, Temporal, Zitadel, TigerBeetle, MailHog)
docker compose up -d
# Format TigerBeetle data file once (first run)
docker compose --profile init run --rm tigerbeetle-init

# Run database migrations
go run ./cmd/migrate

# Start the server
go run ./cmd/main
```

## Project Layout

| Path             | Purpose                                              |
|------------------|------------------------------------------------------|
| `cmd/`           | Entry points (server, CLI, migrations)               |
| `internal/`      | Domain logic grouped by module and slice             |
| `pkg/`           | Shared libraries (DB, NATS, pagination, logging)     |
| `migrations/`    | Database schema migrations (via `pkg/migrate`)       |
| `config/`        | Infrastructure configuration (Caddy, NATS, Temporal) |
| `frontend/`      | Svelte 5 frontend (Vite + Deno)                      |

## Available Commands

```bash
make lint          # vet + workflowcheck + staticcheck + golangci-lint + govulncheck
make fmt           # gofmt + goimports + mdjsonfmt
make test          # unit tests (go test ./...)
make test-db       # DB tests (RUN_DB_TESTS=1, -run ^TestDB)
make test-e2e      # e2e tests (RUN_E2E_TESTS=1)
make fe-dev        # Vite dev server (Deno)
make fe-build      # Vite production build
```

## Dependencies

- Go 1.27+
- Caddy 2.11 (reverse proxy, `127.0.0.1:8090` → `app:8080`)
- PostgreSQL 18
- NATS 2.14 with JetStream
- Temporal 1.29
- Zitadel 4.17 (self-hosted, OIDC/IAM) — https://zitadel.com/self-hosted
- TigerBeetle 0.17 (financial transactions DB) — https://docs.tigerbeetle.com/operating/deploying/docker/
- MailHog (SMTP testing) — https://hub.docker.com/r/mailhog/mailhog

## Stack — Infrastructure Services

Run `docker compose config` to validate `compose.yaml`.

| Service | Image | Ports (host→container) | Purpose |
|---------|-------|------------------------|---------|
| `caddy` | `caddy:2.11.4` | `127.0.0.1:8090:80` | Reverse proxy → `app:8080` (`/health`, gzip, stdout logs) |
| `app` | `golang:1.27.0-alpine` (multi-stage, `frontend/dist` embedded) | — (`expose 8080` only) | Go API + embedded Svelte frontend; healthcheck `GET /health` |
| `postgres` | `postgres:18.4-trixie` | `127.0.0.1:5432:5432` | Primary DB |
| `temporal-postgresql` | `postgres:18.4-trixie` | — | Temporal DB |
| `nats` | `nats:2.14.2-alpine3.22` | `127.0.0.1:4222:4222` | JetStream |
| `temporal` | `temporalio/auto-setup:1.29.7` | `127.0.0.1:7233:7233` | Workflows |
| `zitadel` | `ghcr.io/zitadel/zitadel:v4.17.1` | `127.0.0.1:8080:8080` | IAM/OIDC |
| `zitadel-login` | `ghcr.io/zitadel/zitadel-login:v4.17.1` | `127.0.0.1:3001:3000` | Login UI (Next.js) |
| `tigerbeetle` | `ghcr.io/tigerbeetle/tigerbeetle:0.17.9` | `127.0.0.1:3000:3000` | Ledger DB |
| `tigerbeetle-init` | `ghcr.io/tigerbeetle/tigerbeetle:0.17.9` | — (profile `init`) | One-off format `0_0.tigerbeetle` |
| `mailhog` | `mailhog/mailhog:v1.0.1` | `127.0.0.1:1025:1025` / `127.0.0.1:8025:8025` | SMTP catch-all |

Public entry is `http://localhost:8090` (Caddy). `app` is not published directly; Caddy waits for `app` healthy (`wget /health`) and Caddy itself exposes `/health` for host checks. Config at `config/caddy/Caddyfile`.

## Authentication and notifications

The browser uses a backend-for-frontend OIDC flow: it navigates to `GET /auth/login`,
the Go server handles the Zitadel callback, and the SPA uses only the opaque
PostgreSQL-backed session cookie. Configure the confidential Zitadel Web application
and the `AUTH_*` variables in `.env` before setting `AUTH_ENABLED=true`; the exact
callback URLs must be registered in Zitadel. The SPA must never receive Zitadel tokens.

`internal/auth/loginui` (routes under `/auth/session/*`) lets the Svelte SPA render its
own login form instead of Zitadel's hosted/`zitadel-login` UI, driving Zitadel's Session
API directly. It authenticates its own server-to-server calls with the same
IAM_LOGIN_CLIENT-scoped PAT `zitadel-login` uses (`AUTH_LOGIN_CLIENT_PAT_PATH`, read from
the shared `zitadel-bootstrap` volume, which `app` now also mounts read-only). See
`internal/auth/loginui/README.md`.

`/webhooks/*` is intentionally not proxied by Caddy. To enable Zitadel HTTP notification
delivery, set a strong `WEBHOOKS_ZITADEL_PATH_SECRET`, then configure the Zitadel endpoint
as `http://app:8080/webhooks/zitadel/notifications/<secret>` on the private Compose network.

Volumes: `zitadel-bootstrap` (shared PATs between zitadel, zitadel-login, and app — see `internal/auth/loginui/README.md`), `tigerbeetle-data` (ledger), `caddy-data` / `caddy-config` (Caddy persistence).

TigerBeetle requires `seccomp=unconfined` + `IPC_LOCK` and `--cache-grid`. See `compose.yaml:168`.

Zitadel bootstrap: `MasterkeyNeedsToHave32Characters` (dev only), `ZITADEL_EXTERNALDOMAIN=localhost`, DB `postgres` via `postgres:5432`. Console at `http://localhost:8080/ui/console`, Login UI at `http://localhost:3001/ui/v2/login`.

MailHog: SMTP `localhost:1025`, UI/API `http://localhost:8025`.

## Stack — Go Dependencies

Official clients (latest, pinned in `go.mod`, anchored by `pkg/stack/stack.go` to survive `go mod tidy`):

| Library | Import | Version | Docs |
|---------|--------|---------|------|
| Zitadel Go SDK | `github.com/zitadel/zitadel-go/v3` | `v3.29.3` | https://github.com/zitadel/zitadel-go |
| TigerBeetle Go client | `github.com/tigerbeetle/tigerbeetle-go` | `v0.17.9` | https://github.com/tigerbeetle/tigerbeetle-go / https://docs.tigerbeetle.com/coding/clients/go/ |
| go-mail | `github.com/wneessen/go-mail` | `v0.8.1` | https://github.com/wneessen/go-mail |

Install/update:

```bash
go get github.com/zitadel/zitadel-go/v3@latest
go get github.com/tigerbeetle/tigerbeetle-go@latest
go get github.com/wneessen/go-mail@latest
```

`pkg/stack/stack.go:1` blank-imports them so `go mod tidy` does not prune them while no slice uses them yet.

## Environment

- `.env` — app server (copy from `.env.example`): `SERVER_BIND_ADDR`, `POSTGRES_*`, `NATS_URL`, `TEMPORAL_HOST`, `LOGGER_*`, `AUDIT_LOG_TIMEOUT`, `NOTIFIER_*`
- `.env.compose` — compose overrides (`POSTGRES_HOST=postgres`, `NATS_URL=nats://nats:4222`, `TEMPORAL_HOST=temporal:7233`)
- `config/caddy/Caddyfile` — Caddy reverse proxy (`:80` → `app:8080`, gzip, stdout logs)

Relevant env for new stack (see `compose.yaml`): `ZITADEL_*`, `TIGERBEETLE_ADDRESS`, `MAILHOG_*`. Zitadel uses `MasterkeyNeedsToHave32Characters` in dev; rotate for prod.

Health:

- `GET /health` → `{"status":"ok","commit":"<short-sha>"}` (`pkg/githash`, `internal/health/get`) — liveness, always 200
- `GET /ready` → per-dependency probes (postgres, NATS, Temporal) with 5s timeout — `internal/health/ready`

Built Docker image (`Dockerfile`) is `golang:1.27.0-alpine` builder + `alpine:3.24` runtime, `deno task build` for frontend, `GOOS=linux CGO_ENABLED=0 -mod=vendor`.
