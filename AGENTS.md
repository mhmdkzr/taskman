## Stack

### Backend

- Go
- NATS (with JetStream)

---

## Vertical Slices

Most product behavior is organized as vertical slices under `internal/<module>/<feature>/...`.
Each slice is a Go package focused on one feature or workflow. It should keep its HTTP,
domain, event, and documentation code close together unless the repo already has a more
specific package pattern for that module.

## Modules

Related slices are grouped into modules, which are domain bounded contexts.

Module-level shared types normally live in `internal/<module>/types.go`. Domain newtypes
usually implement `MarshalJSON`/`UnmarshalJSON` for serialization and `String` for display.

Shared runtime dependencies live in `internal/app`. `app.App` contains `Deps`, `Cfg`,
and `Mux`; `Deps` contains runtime dependencies including NATS and JetStream.

Supporting packages live in `pkg/`.

## Registration Pattern

Registration usually follows a two-layer delegation pattern:

1. **Module-level** (`internal/<module>/register/`): Aggregation packages expose `RegisterRoutes`, delegating to constituent slices.
2. **Root-level** (`internal/register/`): Top-level aggregation lives in `routes.go`.

`internal/process/start.go` calls `RegisterRoutes`.

## HTTP Conventions

- Public API slices register routes via `routes.Handle(a, method, path, handler)` (see `internal/routes`).
- Route patterns use Go `net/http` method patterns: `Route.Method` plus `Route.Path`, with path parameters read through `r.PathValue(...)`.
- `app.App.Cfg.Server.BasePath` is applied centrally by `App.RegisterRoutes`; route definitions should remain module-relative, usually beginning with `/`.
- Query slices should normally use `GET`.
- Command slices should normally use `POST`, `PUT`, or `DELETE` and expose HTTP action routes whose last path segment is the action when the resource is not fully described by the method alone.
- Handlers should parse path/query/body input into typed `Request` values via a `requestFromHTTP(r *http.Request)` function, then call a business logic function, write success with `jsonresp.WriteJSON`, and write errors with `jsonresp.WriteHTTPError`.
- Handlers should map domain and validation errors to appropriate HTTP status codes via a local `httpStatusForError(err) int` function.

## NATS JetStream

- Each module that publishes events owns a `streams.go` file with a `CreateStreams(ctx, js)` function that calls `js.CreateOrUpdateStream`.
- Subject names follow a dot-separated hierarchical convention.
- Event types implement `MsgID() string` for idempotent publishing via the generic `produce.Produce[Event interface{ MsgID() string }](ctx, js, subject, event)` helper.
- Consumption uses `jetstream.Consumer` directly via `js.CreateOrUpdateConsumer` and `consumer.Consume` with manual NAK/ACK handling.

## Agent runtime

The application runs an LLM agent runtime alongside the HTTP server (wired in `internal/process/start.go`). Key packages:

- `internal/agent` — wraps `github.com/zendev-sh/goai` behind agent-owned types; sessions, run/persist/fork, the scheduled-run `Consumer`, and the default tool set + sub-agent `Runner`.
- `internal/events` — every bus message type; each implements `Subject()` and `MsgID()` (deterministic content hash). New event types must implement both.
- `internal/publisher` — sole gateway to the bus; publishes with a `Nats-Msg-Id` dedup header for exactly-once delivery. Owns the `TASKMAN` stream (`agent.>`, `scheduler.>`, MemoryStorage, 24h dedup window) via `CreateStreams`.
- `internal/scheduler` — durable one-time/recurring message delivery backed by SQLite rows.
- `internal/server` — assembles store + scheduler + consumer and answers `run`/`ping` request/reply on `protocol.Subject` (`taskman.request`), one request at a time.
- `internal/protocol` — dependency-free client/server wire contract.
- `internal/store` — SQLite (pure-Go `modernc.org/sqlite`), RW/RO handles, sessions as a shared message chain, fork-by-reference, schema in `migrations/schema.sql` applied idempotently (no numbering; change in place).
- `internal/tools` — only `bash`, `files`, `spawn`, and `telegram` are registered, plus `schedule` (the `agent.run` wire contract only — no scheduling tools). `internal/tools/tools_test.go` asserts the exact tool count.
- `internal/web` — read-only realtime dashboard served by the app HTTP server at `/`: renders tasks/sessions from the store and streams live bus events over SSE (`agent.>`, `scheduler.>`). No control surface.
- `internal/runner` — server-side executor for the `taskman run` command: subscribes to `taskman.command.run` and drives each task through the pipeline. Commands are core-NATS messages (`publisher.PublishCore`/`SubscribeCore`), not stream events.

Config comes from `AGENT_*` env vars (`config.AgentConfig`): `AGENT_PROVIDER_BASE_URL`, `AGENT_PROVIDER_API_KEY` (required), `AGENT_DB_PATH` (default `~/.taskman/taskman.db`), `AGENT_MODEL`, `AGENT_REASONING_EFFORT`, `AGENT_MAX_STEPS`, and optional `AGENT_TELEGRAM_API_KEY`/`AGENT_TELEGRAM_CHANNEL_ID` for the telegram tool. There is no config file; the old `pkg/notifier` is gone.

## Pagination

- List endpoints use `pkg/pagination.Meta` (fields: `Page`, `Size`, `Total`) for paginated responses.
- Use `pagination.Normalize` for defaults and `pagination.Validate` for bounds checking.

---

## Required Slice Documentation

New slices must include a `README.md` file which explains what the slice is, what functionality it provides, how it behaves, and how it is invoked. When touching an existing slice, update its README if present; if the slice lacks one and the change is material, add it. For public API slices, document the HTTP route and include `curl` examples.

---

## Error Handling

- Define domain errors as sentinel `var` values with `errors.New(...)`.
- Use `errors.Is()` and `errors.AsType[T]()` for error checking and unwrapping.
- Wrap errors with context using `fmt.Errorf("context: %w", err)` to provide error chains when useful.
- We almost always should return errors, but if an error is not being explicitly returned, intentionally, the reason should always be explained via a comment and the error **must be logged with `Error` level**. There must be **no silent errors**.
- HTTP handlers write domain errors to the response using `jsonresp.WriteHTTPError(w, status, err)`, which serializes as `{"error": "..."}` via `jsonresp.ResponseError`.

---

## Testing

- If you change `[file].go`, and `[file]_test.go` or other related test files are present, keep them in sync with the behavior you changed.
- If you need to use NATS in a test, use `pkg/natsembed` when an in-process NATS server is enough.
- For e2e tests that run from `testing.T`, prefer `t.Context()` over `context.Background()` so request cancellation is tied to test lifecycle.
- Do NOT use mocks, unless you have checked with user and got a validation for your usecase.
- Tests can load .env files if they need their values (see `pkg/testenv`).
- Integration tests live in `_integration_test.go` files and are gated with `testenv.SkipIfE2ETestsDisabled` (or the network equivalents).

---

## Build and Validation

- After making code changes, run the smallest sensible build/test/vet scope.
- Use `go vet`, and try to build the code so we can catch any compile-time errors. Do not store build artifacts; send them to `/dev/null` when building binaries.
- Use `make lint` for running linters and `make fmt` for formatting.
- For doc-only changes, Go validation is not required.

---

## Documentation

- The documentation in code should be accurate, clear and **up-to-date**, **in sync with code**. This includes the comments in code, module docs, and README.md files. Don't forget to update them when code changes.
- If you're using an external Go library, you can use `go doc` command to read its up-to-date docs and understand how to use it properly.

---

## Logging

- Use `slog` package for logging with proper log level and attrs.

---

## Type System and Design Principles

- Don't introduce `interface{}` or `any` types unless the standard library or a generic API genuinely requires it. If there is a design choice, confirm with the user first.
- **Make invalid state irrepresentable**, when possible.
- Use the type system to **prevent invalid states at compile time**.
- Utilize the newtype pattern with constructors that validate the input and return an error if the input is invalid, so we know all instances of the type are valid.
- Prefer **simple** and **minimal** code. Care about **clarity** of the whole.
- Avoid premature optimization and over-engineering.
- Avoid sycophancy.
- Use current Go syntax and features already supported by this repo's workspace Go version.
- Detect important and critical decision points. When you find a decision point in front of you which you can't know what to do based on your context, **confirm your decisions with user** before taking actions. For simple decisions or decisions that can be made with current context, you don't need to do this.
- Avoid premature abstractions. Don't add a level of indirection unless it actually helps and the indirection is worth the cost of it. Do not use helper functions that don't help reduce complexity and are better inlined.
- Any duration crossing a wire or storage boundary (JSON API fields) uses **raw nanoseconds**, matching Go's `time.Duration` (already an `int64` nanosecond count) exactly — no unit conversion at any layer. Domain code keeps using `time.Duration` natively.

---

## Tooling and Commands

- available cli tools:
  - `rg`
  - `jq`
  - `nats`
- To see project structure, run `tree`.
- Do not modify `go.mod` file directly. Use `go` commands for it, e.g., use `go get` instead of adding dependencies manually.

---

## Tasks (`.tasks/`)

Some packages carry a `<package>/.tasks/` directory of tracked follow-up work (review findings, test gaps, doc drift, features). Each task is a Markdown file with YAML frontmatter (`urgency`/`importance`, `type`, `status`, `tags`, `depends_on`, `where`, `source`, `resolved`/`resolved_at`) plus a short prose body: what the task is, optionally how to do it, why it matters, and what "done" looks like (`## What`/`## How`/`## Why`/`## Done when`, plus `## Resolution` once resolved).

Never hand-write or hand-edit a `.tasks/*.md` file. Use the `taskman` CLI at `scripts/taskman/` (build with `cd scripts/taskman && GOWORK=off go build -o taskman .`) for every operation — `taskman new`/`list`/`show`/`search`/`done`/`drop`/`validate`. See `scripts/taskman/README.md` for the full schema and usage.

---

## Git Policy

- Git access is read-only by default. Do not create branches, commit, amend, rebase, merge, tag, or push unless the user explicitly asks.
- Before committing, always show:
  - what is staged
  - the proposed commit message
  - a short summary of the change
  Then ask the user for approval.
  If user provided a `-y` flag to the prompt (e.g. commit changes -y) you don't need to ask for approval.

- Use conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`, see `notes/resources/conventional-commits.md`).
- Use conventional branch names such as `feat/...`, `fix/...`, `refactor/...`, `docs/...`, `test/...`. Do not use `codex/` prefixes.
- Prefer small, incremental, focused commits. One logical change per commit.
- Prefer package or slice-scoped commits. Do not mix unrelated changes in one commit.
- Do not rewrite git history unless the user explicitly asks. Avoid `--amend`, rebase, and force-push by default.
- Keep the worktree clean. Do not leave unrelated modified files, debug edits, or incidental formatting changes mixed into the work.
- Before asking to commit, ensure the changed code builds and relevant tests pass at the smallest sensible scope.
- Do not commit generated files, temporary files, local-only config, or anything containing secrets.
- When code changes require doc updates, include the relevant doc updates in the same commit when practical.
- Run `make lint` and `make fmt` before committing.
- Don't add "Co-Authored-By: Claude ..." line to commit messages.

--- 

## Deployment

Use semantic versioning for release tags (see `notes/resources/semantic-versioning.md`).

--- 

- Be concise and task-focused.
