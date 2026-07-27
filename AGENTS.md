## Overview

Go backend (vertical slice architecture) and frontend are both at the project root. Go code lives under `cmd/`, `internal/`, `pkg/`, etc. Frontend is a plain Svelte (not SvelteKit) project at `frontend/`, built with Vite + Deno.

## Frontend UI Libraries

- **Svelte 5** — component framework
- **Tailwind CSS v4** — utility-first CSS
- **shadcn-svelte** — component library / registry (Rhea style), configured in `frontend/components.json`
- **Phosphor Icons** — icon library (via `phosphor-svelte`)
- **Lucide Icons** — icon library (via `@lucide/svelte`)
- **Geist** — font (via `@fontsource-variable/geist`)
- **Source Sans 3** — font (via `@fontsource-variable/source-sans-3`)
- **tailwind-merge** + **clsx** — class merging utilities
- **tailwind-variants** — component variants
- **tw-animate-css** — animation utilities

---

## Embedding

The frontend static build output (`frontend/dist/`) is embedded in the Go binary at compile time via `dist.go` (`package app`; exports `app.FS embed.FS`). In development, run the Vite dev server separately with `make fe-dev`. In production, the Go binary serves the embedded assets.

## Vertical Slices

Most product behavior is organized as vertical slices under `internal/<module>/<feature>/...`.
Each slice is a Go package focused on one feature or workflow. It should keep its HTTP,
domain, persistence, Temporal, event, and documentation code close together unless the repo
already has a more specific package pattern for that module.

## Modules

Related slices are grouped into modules, which are domain bounded contexts.
Module-level shared types normally live in `internal/<module>/types.go`.
Shared runtime dependencies live in `internal/app`. `app.App` contains `Deps`, `Cfg`,
and `Mux`; `Deps` contains PostgreSQL, NATS, JetStream, and Temporal.

## Registration Pattern

Registration usually follows a three-layer delegation pattern:

1. **Slice-level**: A slice exposes registration helpers such as `RegisterWorkflows(w worker.Worker, a app.App)` and `RegisterActivities(w worker.Worker, a app.App)`. Some slices put these helpers in `register.go`; others register directly from `workflow.go`, `activity.go`, or a module register package. Follow the nearby pattern.

2. **Module-level** (`internal/<module>/register/`): Aggregation packages expose `RegisterRoutes`, `RegisterWorkflows`, and/or `RegisterActivities`, each delegating to constituent slices.

3. **Root-level** (`internal/register/`): Top-level aggregation is split across `routes.go`, `workflows.go`, `activities.go`, and `events.go`. `internal/process/start.go` calls `RegisterRoutes`, `RegisterActivities`, and `RegisterWorkflows`.

## HTTP Conventions

- Public API slices register routes via `a.Handle(method, path, handler)` (see `internal/app/router.go`).
- Route patterns use Go `net/http` method patterns: `Route.Method` plus `Route.Path`, with path parameters read through `r.PathValue(...)`.
- `app.App.Cfg.Server.BasePath` is applied centrally by `App.RegisterRoutes`; route definitions should remain module-relative, usually beginning with `/`.
- Query slices should normally use `GET`.
- Command slices should normally use `POST`, `PUT`, or `DELETE` and expose HTTP action routes whose last path segment is the action when the resource is not fully described by the method alone.
- Handlers should parse path/query/body input into typed `Request` values via a `requestFromHTTP(r *http.Request)` function, then call a business logic function, write success with `app.WriteJSON`, and write errors with `app.WriteHTTPError`.
- Handlers should map domain, validation, and Temporal errors to appropriate HTTP status codes via a local `httpStatusForError(err) int` function.
- NATS and JetStream are still valid for internal messaging, event publication, consumers, and streaming integrations.

## JetStream Events

- Each module that publishes events owns a `streams.go` file with a `CreateStreams(ctx, js)` function that calls `js.CreateOrUpdateStream`. Streams use `Duplicates: 24 * time.Hour` for deduplication.
- Subject names follow a dot-separated hierarchical convention.
- Event types implement `MsgID() string` for idempotent publishing via the generic `app.Produce[Event interface{ MsgID() string }](ctx, js, subject, event)` helper.
- Consumption uses `jetstream.Consumer` directly via `js.CreateOrUpdateConsumer` and `consumer.Consume` with manual NAK/ACK handling.

## Pagination

- List endpoints use `pkg/pagination.Meta` (fields: `Page`, `Size`, `Total`) for paginated responses.
- Use `pagination.Normalize` for defaults and `pagination.Validate` for bounds checking.

---

## Required Slice Documentation

New slices must include a `README.md` file which explains what the slice is, what functionality it provides, how it behaves, and how it is invoked. When touching an existing slice, update its README if present; if the slice lacks one and the change is material, add it. For public API slices, document the HTTP route and include `curl` examples. For Temporal-backed slices, include Temporal CLI examples when useful.

---

## Error Handling

- Define domain errors as sentinel `var` values with `errors.New(...)`.
- Use `errors.Is()` and `errors.As()` for error checking and unwrapping.
- Wrap errors with context using `fmt.Errorf("context: %w", err)` to provide error chains when useful.
- We almost always should return errors, but if an error is not being explicitly returned, intentionally, the reason should always be explained via a comment and the error **must be logged with `Error` level**. There must be **no silent errors**.
- HTTP handlers write domain errors to the response using `app.WriteHTTPError(w, status, err)`, which serializes as `{"error": "..."}` via `app.ErrorResponse`.

---

## Code and Test Sync

- If you change `[file].go`, and `[file]_test.go` or other related test files are present, keep them in sync with the behavior you changed.

---

## Testing Rules

- If you need to use NATS in a test, use `pkg/natsembed` when an in-process NATS server is enough.
- For e2e tests that run from `testing.T`, prefer `t.Context()` over `context.Background()` so request cancellation is tied to test lifecycle.
- For testing DB-backed slices, use a temporary PostgreSQL database:
  - create temp DB
  - run migrations
  - seed minimal fixture rows
  - run slice logic
  - assert
  - drop DB in cleanup
- Prefer the repo's `pkg/testdb` and Testcontainers pattern for PostgreSQL-backed tests:
  - start `postgres.Run(...)` with a disposable container image
  - register cleanup with `testcontainers.CleanupContainer`
  - use the container connection string with `sslmode=disable`
  - run migrations, then reopen/ping the database before seeding fixtures
- Do NOT use mocks, unless you have checked with user and got a validation for your usecase.
- Tests can load .env files if they need their values (see pkg/testenv for it).

---

## Build and Validation

- After making code changes, run the smallest sensible build/test/vet scope.
- All Go code lives under the project root. Run Go commands from the root, e.g., `go vet ./...`.
- Use `go vet`, and try to build the code so we can catch any compile-time errors. Do not store build artifacts; send them to `/dev/null` when building binaries.
- For doc-only changes, Go validation is not required.

---

## Database

- You should not write to database directly unless you have a good reason to do so, in that case, **confirm with user**. Use the system endpoints instead, keep direct db access read-only and for debugging when API wouldn't be enough.

### Repository Pattern

All database access must be wrapped in private functions whose only job is to take a `*sql.DB` (or `*sql.Tx`) and interact with the database. They would all be in `repo.go` files, and their tests in `repo_test.go` files.

---

## Documentation Accuracy

- The documentation in code should be accurate, clear and **up-to-date**, **in sync with code**. This includes the comments in code, module docs, and README.md files. Don't forget to update them when needed.
- If you're using an external Go library, you can use `go doc` command to read its up-to-date docs and understand how to use it properly.

---

## Logging

- Use `slog` package for logging with proper log level and attrs.

---

## Type System and Design Principles

- Don't introduce `interface{}` or `any` types unless the standard library or a generic API genuinely requires it. If there is a design choice, confirm with the user first.
- **Make invalid state irrepresentable**, when possible.
- Use the type system to **prevent invalid states at compile time**.
- Prefer **simple** and **minimal** code. Care about **clarity** of the whole.
- Avoid premature optimization and over-engineering.
- Avoid sycophancy.
- Use current Go syntax and features already supported by this repo's workspace Go version.
- Detect important and critical decision points. When you find a decision point in front of you which you can't know what to do based on your context, **confirm your decisions with user** before taking actions. For simple decisions or decisions that can be made with current context, you don't need to do this.
- Avoid premature abstractions. Don't add a level of indirection unless it actually helps and the indirection is worth the cost of it. Don't use helper functions that don't help reduce complexity and are better inlined.

---

## Tooling and Commands

- available cli tools:
  - `rg`
  - `jq`
  - `psql` (available through postgres docker container, not on the host)
  - `nats`
  - `temporal`
  - `workflowcheck`
- To see project structure, run `tree`.
- All Go code lives under the project root. Run Go commands from the root, e.g., `go vet ./...`.
- Do not modify `go.mod` file directly. Use `go` commands for it, e.g., use `go get` instead of adding dependencies manually.

---

## Temporal Rules

- When working with Temporal, remember that temporal works under the assumption that workflows are deterministic and side-effect free, and activities are idempotent. Make sure this is true, otherwise it is a bug.
- Workflows should validate inputs at the start, returning `temporal.NewNonRetryableApplicationError(err.Error(), "ValidationError", nil)` for invalid inputs.

---

## Git Policy

- Git access is read-only by default. Do not create branches, commit, amend, rebase, merge, tag, or push unless the user explicitly asks.
- Before committing, always show:
  - what is staged
  - the proposed commit message
  - a short summary of the change
  Then ask the user for approval.
  If user provided a `-y` flag to the prompt (e.g. commit changes -y) you don't need to ask for approval.

- Use conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`).
- Use conventional branch names such as `feat/...`, `fix/...`, `refactor/...`, `docs/...`, `test/...`. Do not use `codex/` prefixes.
- Prefer small, incremental, focused commits. One logical change per commit.
- Prefer package- or slice-scoped commits. Do not mix unrelated changes in one commit.
- Separate refactors from behavior changes when practical.
- Do not rewrite git history unless the user explicitly asks. Avoid `--amend`, rebase, and force-push by default.
- Keep the worktree clean. Do not leave unrelated modified files, debug edits, or incidental formatting changes mixed into the work.
- Before asking to commit, ensure the changed code builds and relevant tests pass at the smallest sensible scope.
- Do not commit generated files, temporary files, local-only config, or anything containing secrets.
- When code changes require doc updates, include the relevant doc updates in the same commit when practical.
- Run `gofmt -w .` and check the code via `go vet ./...` before committing.
