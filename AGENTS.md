## Stack

- Go 1.27+
- SQLite-backed, event-sourced task store (`internal/task/store`, via `modernc.org/sqlite`): a
  task's current state is never stored directly, only derived by replaying its events through
  `task.Apply`
- `git` on `PATH` (taskman shells out through `internal/git` to read a worktree's current commit)
- `urfave/cli` v3 for the CLI frontend, `modelcontextprotocol/go-sdk` for the MCP frontend

Taskman is a one-shot process that validates and records the state transitions reported to it, either as a CLI command or (via `taskman mcp`) as an MCP/stdio tool call.

---

## Vertical Slices

Every `taskman <command>` is a vertical slice: one package under `internal/commands`. The two
review-bearing stages each get their own grouping tree -
`internal/commands/specification/review/{agent,human}/{approved,rejected}` and
`internal/commands/implementation/review/{agent,human}/{approved,rejected}` - four fully
independent leaf packages per stage (`specification review agent approved`, `implementation
review human rejected`, etc.), each backed by its own dedicated `task.TaskEvent` type
(`SpecificationReviewAgentApproved`, `ImplementationReviewHumanRejected`, ...): stage and reviewer
type are baked into the event's Go type itself, not inferred from the task's state at apply time,
so nothing is shared between the two stages' review commands even though a couple of them (the
agent-review pair) happen to have identical bodies. There is no intermediate grouping command
otherwise - every slice's `Command()` mounts directly on the `taskman` root
(`internal/cli/cli.go`), except `specification`/`implementation`'s own `review/{agent,human}`
nesting. A slice keeps its domain logic
(`<name>.go`), its frontends - `cmd.go` (CLI, exports `Command()`) and, where wired up, `mcp.go`
(MCP, exports `RegisterMCP()`) - and documentation (`README.md`) close together. A slice's
application function never imports either frontend package; `cmd.go` and `mcp.go` both call
straight into it, so the two frontends stay two thin, independent callers of the same logic
rather than one wrapping the other. Workflow commands are imperative shells: they validate input,
obtain external facts, construct a typed `task.TaskEvent`, and apply it through the store's
`Append`, which validates via the pure `task.Apply` before persisting. An application function's
entire caller-supplied input - including the task id itself, since MCP has no positional-argument
concept the way a CLI does - is one `Request` struct defined in `<name>.go` (not duplicated per
frontend): `json`/`jsonschema` struct tags make it usable as-is for both `cmd.go` (built
field-by-field from flags) and `mcp.go` (passed directly as the tool's typed input). Where
`Request` has anything worth checking (a required field, a non-empty id), it gets a
`func (r Request) validate() error` method, called once at the top of the domain function -
shared automatically by both frontends rather than checked twice or only in one. Runtime deps a
frontend doesn't get from the caller (`*store.Store`, `*git.Client`) stay separate function
parameters, not `Request` fields, since MCP binds them once at server startup rather than per
call.

## Shared Packages

- `internal/task` - the pure functional core: the `Task` aggregate, its append-only
  `StateHistory` (current state is `Task.State()`, its last entry - never a separate mutable
  field), the closed set of `TaskEvent` types, the declarative compiled workflow (`workflow.go`),
  `Apply`, and `Instruction` (the pure per-state projection of what should happen next). It
  performs no filesystem, Git, clock, logging, CLI, or MCP operations - every event carries its
  own `At`, supplied by the caller. `Apply` clones its input before reducing an event, so callers
  never observe partial mutation.
- `internal/task/store` - SQLite-backed, event-sourced persistence (`Store`, opened once via
  `Open`): `Create`, `Read` (replays a task's events through `task.Apply`), `Append` (validates one
  more event via `task.Apply` inside a single transaction before persisting it - a rejected event
  is never written), `Delete` (the sole non-append-only operation: it permanently removes a task
  and its event log by identity, without replaying the log), and `List`.
- `internal/git` - the imperative Git adapter used by command shells: currently just
  `ReadCommit`, reading a worktree's current commit hash/message.
- `internal/utils` - CLI-only plumbing shared by every slice's `cmd.go`: building a
  `*git.Client`/`*store.Store` from root flags, parsing repeated `key=value` flags, rendering
  output (`--json` envelope, `--md` Markdown document, or human-readable summary), and mapping
  errors to exit codes.

There is no `pkg/`; everything shared lives under `internal/`.

## Command Assembly Pattern

1. **Slice-level**: each slice's `cmd.go` exports a `Command()` func returning a `*cli.Command`;
   a slice wired up for MCP also has `mcp.go` exporting `RegisterMCP()`, which adds that slice's
   tool to an `*mcp.Server`.
2. **Stage-level**: `internal/commands/specification/cmd.go` mounts `review`, which mounts
   `agent`/`human`, each of which mounts its own `approved`/`rejected` children -
   `internal/commands/implementation` mirrors this exactly. These are the only nested grouping
   commands. `internal/mcp/server.go` mounts every wired-up slice's `RegisterMCP()` into the MCP
   server the same way (flat - MCP tools aren't nested, so the four leaves under each stage get
   distinct, self-describing tool names like `task_specification_review_agent_approved`), sharing
   one `*store.Store` and `*git.Client` across every tool call for the server's lifetime.
3. **Root-level**: `internal/cli/cli.go`'s `rootCommand` mounts every slice's `Command()` -
   including the two review-stage grouping commands, `skill.Command()`, and `mcp.Command()` (from
   `internal/commands/mcp`, which opens a `*store.Store` from `--db`, calls
   `internal/mcp.NewServer`, and serves it over stdio) - directly under the `taskman` root with
   the root flags, and wires `initLogger` (defined in the same file) as its `Before` hook.

## CLI Conventions

- A slice's `cmd.go` `Action` parses flags into the request its domain function expects, opens a
  `*store.Store` via `utils.StoreFrom` (closed with `defer`), calls the application function,
  renders success via `internal/utils` (`utils.PrintTask`/`utils.PrintJSON`), and returns errors
  through `utils.Fail`.
- Every flag has a `Usage` string written for someone who only has the compiled binary - no
  references to files or paths in this repo. `--id` is always a flag, never a positional
  argument, so the same `Request` struct binds identically for MCP.
- Slices must not import `internal/cli` (it imports them, so that would cycle) - shared helpers
  go in `internal/utils`, which depends on `internal/task`, `internal/task/store`, `internal/git`,
  and `urfave/cli` only.

---

## Required Slice Documentation

New slices must include a `README.md` file which explains what the slice is, what functionality it provides, how it behaves, and how it is invoked. When touching an existing slice, update its README if present; if the slice lacks one and the change is material, add it.

---

## Error Handling

- Define domain errors as sentinel `var` values with `errors.New(...)`, and structured errors as
  error types (e.g. `*task.InvalidTransitionError`).
- Use `errors.Is()` and `errors.AsType[T]()` for error checking and unwrapping.
- Wrap errors with context using `fmt.Errorf("context: %w", err)` to provide error chains when useful.
- We almost always should return errors, but if an error is not being explicitly returned, intentionally, the reason should always be explained via a comment and the error **must be logged with `Error` level**. There must be **no silent errors**.
- CLI slices return errors; the command's `Action` hands them to `utils.Fail`, which maps
  domain errors to exit codes (see `utils.ExitCode`).

---

## Testing

- If you change `[file].go`, and `[file]_test.go` or other related test files are present, keep them in sync with the behavior you changed.
- Every slice has unit tests in `_test.go` files running under plain `go test ./...`.
- `internal/cli/cli_integration_test.go` drives the real command tree in-process (with
  `cli.OsExiter` stubbed) against throwaway git repositories; keep it in sync with command behavior.
- For tests that run from `testing.T`, prefer `t.Context()` over `context.Background()` so request cancellation is tied to test lifecycle.
- Do NOT use mocks, unless you have checked with user and got a validation for your usecase.

---

## Build and Validation

- After making code changes, run the smallest sensible build/test/vet scope.
- Use `go vet`, and try to build the code so we can catch any compile-time errors. Do not store build artifacts; send them to `/dev/null` when building binaries.
- Use `make lint` for running linters and `make fmt` for formatting; `make test` runs the full suite.
- For doc-only changes, Go validation is not required.

---

## Documentation

- The documentation in code should be accurate, clear and **up-to-date**, **in sync with code**. This includes the comments in code, module docs, and README.md files. Don't forget to update them when code changes.
- If you're using an external Go library, you can use `go doc` command to read its up-to-date docs and understand how to use it properly.

---

## Logging

- Use `slog` package for logging with proper log level and attrs. The root `--log-level`/`--log-format` flags configure it (see `initLogger` in `internal/cli/cli.go`).

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

---

## Tooling and Commands

- available cli tools:
  - `rg`
  - `jq`
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

- Use conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`).
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

Use semantic versioning for release tags.

--- 

- Be concise and task-focused.
- Go v1.27 has built-in UUID library. Use it over external libraries.
