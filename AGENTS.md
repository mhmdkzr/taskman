## Stack

- Go
- SQLite
- Templ
- Datastar

---

## Vertical Slices

Most product behavior is organized as vertical slices under `internal/<module>/<feature>/...`.
Each slice is a Go package focused on one feature or workflow. It should keep its HTTP,
domain, persistence, event, and documentation code close together unless the repo
already has a more specific package pattern for that module.

## Modules

Related slices are grouped into modules, which are domain bounded contexts.

Module-level shared types normally live in `internal/<module>/types.go`. Domain newtypes
usually implement `Scan`/`Value` for database, `MarshalJSON`/`UnmarshalJSON` for serialization, and `String` for display. 

Shared runtime dependencies live in `internal/app`. `app.App` contains `Deps`, `Cfg`,
and `Mux`; `Deps` contains runtime dependencies including SQLite.

Supporting packages live in `pkg/`.

## Registration Pattern

Registration usually follows a three-layer delegation pattern:

1. **Slice-level**: A slice exposes registration helpers in `register.go` when needed.
2. **Module-level** (`internal/<module>/register/`): Aggregation packages delegate to constituent slices.
3. **Root-level** (`internal/app/register/`): Top-level aggregation is split by concern.

`internal/app/process/start.go` calls the root registration helpers.

## HTTP Conventions

- Public API slices register routes via `routes.Route` descriptors (`Method`, `Path`, `Handler`) passed to `routes.RegisterRoutes(a, ...)` (see `internal/app/routes`).
- Route patterns use Go `net/http` method patterns: `Route.Method` plus `Route.Path`, with path parameters read through `r.PathValue(...)`.
- `app.App.Cfg.Server.BasePath` is applied centrally by `App.RegisterRoutes`; route definitions should remain module-relative, usually beginning with `/`.
- Query slices should normally use `GET`.
- Command slices should normally use `POST`, `PUT`, or `DELETE` and expose HTTP action routes whose last path segment is the action when the resource is not fully described by the method alone.
- Handlers should parse path/query/body input into typed `Request` values via a `requestFromHTTP(r *http.Request)` function, then call a business logic function, write success with `jsonresp.WriteJSON`, and write errors with `jsonresp.WriteHTTPError`.
- Handlers should map domain and validation errors to appropriate HTTP status codes via a local `httpStatusForError(err) int` function.

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
- For e2e tests that run from `testing.T`, prefer `t.Context()` over `context.Background()` so request cancellation is tied to test lifecycle.
- For testing DB-backed slices, use an in-memory SQLite database and run the schema migrations before seeding fixtures.
- Do NOT use mocks, unless you have checked with user and got a validation for your usecase.
- Tests can load .env files if they need their values (see `pkg/testenv`).
- Test files follow a `_test.go` / `_integration_test.go` split:
  - Pure unit tests live in `_test.go` files and run under plain `go test ./...`.
  - Integration tests (including DB-backed tests) live in `_integration_test.go` files (e.g. `repo_integration_test.go` for repository tests) and are gated with `testenv.SkipIfDBTestsDisabled` (or the network/e2e equivalents).
  - Tests that are primarily about database behavior must be named with a `TestDB` prefix so they can be run selectively via `go test -run '^TestDB'` (see `make test-db`). Since `TestDB` implies integration, do not also append `_Integration` to their names. 

---

## Build and Validation

- After making code changes, run the smallest sensible build/test/vet scope.
- Use `go vet`, and try to build the code so we can catch any compile-time errors. Do not store build artifacts; send them to `/dev/null` when building binaries.
- Use `make lint` for running linters and `make fmt` for formatting.
- For doc-only changes, Go validation is not required.

---

## Database

- You should not write to database directly unless you have a good reason to do so, in that case, **confirm with user**. Use the system endpoints instead, keep direct db access read-only and for debugging when API wouldn't be enough.

### Repository Pattern

All database access must be wrapped in private functions whose only job is to take a `*sql.DB` (or `*sql.Tx`) and interact with the database. They would all be in `repo.go` files, and their tests in `repo_integration_test.go` files.

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
- Any duration crossing a wire or storage boundary (SQLite integer columns or JSON API fields) uses **raw nanoseconds**, matching Go's `time.Duration` exactly. Domain code keeps using `time.Duration` natively.

---

## Tooling and Commands

- available cli tools:
  - `rg`
  - `jq`
  - `psql`
  - `nats`
- To see project structure, run `tree`.
- Do not modify `go.mod` file directly. Use `go` commands for it, e.g., use `go get` instead of adding dependencies manually.

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
- Go v1.27 has built-in UUID library. Use it over external libraries.
