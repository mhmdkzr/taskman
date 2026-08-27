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

| Path             | Purpose                                         |
|------------------|--------------------------------------------------|
| `cmd/`           | Entry points (server, CLI, migrations)           |
| `internal/`      | Domain logic grouped by module and slice         |
| `pkg/`           | Shared libraries (DB, NATS, pagination, logging) |
| `migrations/`    | Database schema migrations                       |
| `config/`        | Infrastructure configuration (NATS, Temporal)    |
| `frontend/`      | Svelte 5 frontend (Vite + Deno)                  |

## Available Commands

```bash
make lint       # fmt + vet + staticcheck + govulncheck + test
make test       # run all tests
make fe-dev     # run Vite dev server
```

## Dependencies

- Go 1.26+
- PostgreSQL 18
- NATS 2.14 with JetStream
- Temporal 1.29
- Zitadel 4.17 (self-hosted, OIDC/IAM) — https://zitadel.com/self-hosted
- TigerBeetle 0.17 (financial transactions DB) — https://docs.tigerbeetle.com/operating/deploying/docker/
- MailHog (SMTP testing) — https://hub.docker.com/r/mailhog/mailhog

## Stack — Infrastructure Services

All images in `compose.yaml` are pinned to `tag@sha256:digest`. Run `docker compose config` to validate.

| Service | Image | Ports (host→container) | Purpose |
|---------|-------|------------------------|---------|
| `postgres` | `postgres:18.4-trixie@sha256:29ee7bb30d804447dc9a91fd0d74322ae1dc3a4072cc6346f70a5ed6e783b565` | `127.0.0.1:5432:5432` | Primary DB |
| `temporal-postgresql` | `postgres:18.4-trixie@sha256:29ee7bb30d804447dc9a91fd0d74322ae1dc3a4072cc6346f70a5ed6e783b565` | — | Temporal DB |
| `nats` | `nats:2.14.2-alpine3.22@sha256:b039b46715673a9436989cfc49dde04e6bd57e205347478a58214789baf5efdc` | `127.0.0.1:4222:4222` | JetStream |
| `temporal` | `temporalio/auto-setup:1.29.7@sha256:f14912b699cf73015ad5c4fc18d522d4b014db90e794039214dfb7c022c2644f` | `127.0.0.1:7233:7233` | Workflows |
| `zitadel` | `ghcr.io/zitadel/zitadel:v4.17.1@sha256:3ac6910685d48f32481f01f45e3e6215efe5a9df2c069591b481e9a101712db5` | `127.0.0.1:8080:8080` | IAM/OIDC |
| `zitadel-login` | `ghcr.io/zitadel/zitadel-login:v4.17.1@sha256:8035df2409afb35a3999482ee98e453261715f98d47e4b62e948e4a1ddf4345f` | `127.0.0.1:3001:3000` | Login UI (Next.js) |
| `tigerbeetle` | `ghcr.io/tigerbeetle/tigerbeetle:0.17.9@sha256:48f623f9c1e9b6cc44d77ca93634595ae99cce3246ded418763eb1a62eee45e9` | `127.0.0.1:3000:3000` | Ledger DB |
| `tigerbeetle-init` | `ghcr.io/tigerbeetle/tigerbeetle:0.17.9@sha256:48f623f9c1e9b6cc44d77ca93634595ae99cce3246ded418763eb1a62eee45e9` | — (profile `init`) | One-off format `0_0.tigerbeetle` |
| `mailhog` | `mailhog/mailhog:v1.0.1@sha256:8d76a3d4ffa32a3661311944007a415332c4bb855657f4f6c57996405c009bea` | `127.0.0.1:1025:1025` / `127.0.0.1:8025:8025` | SMTP catch-all |

Volumes: `zitadel-bootstrap` (shared PAT between zitadel and zitadel-login), `tigerbeetle-data` (ledger).

TigerBeetle requires `seccomp=unconfined` + `IPC_LOCK` and `--cache-grid`. See `compose.yaml:143`.

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

- `.env` — app server (copy from `.env.example`)
- `.env.compose` — compose overrides (`POSTGRES_HOST=postgres`, `NATS_URL=nats://nats:4222`, `TEMPORAL_HOST=temporal:7233`)

Relevant env for new stack (see `compose.yaml:78`): `ZITADEL_*`, `TIGERBEETLE_ADDRESS`, `MAILHOG_*`. Zitadel uses `MasterkeyNeedsToHave32Characters` in dev; rotate for prod.
