## Stack

- Go 1.27+
- File-backed YAML task store (`.tasks/*.yaml`) with per-task file locking
- `git` on `PATH` (taskman shells out to it for worktrees and reading commits)
- `urfave/cli` v3

No database, no HTTP server, no MCP - taskman is a one-shot CLI that validates and records the
state transitions reported to it. See `notes/design/design.md` for the full design.

---

## Vertical Slices

Every `taskman task <command>` is a vertical slice: one package under `internal/cli/task`, with
the review stage nested as `internal/cli/task/review/{record,approve,reject}`. A slice keeps its
CLI (`cmd.go`), domain logic (`<name>.go`), prompt templates (`prompt.go`/`prompt.md` where the
command dispatches an agent), and documentation (`README.md`) close together.

## Shared Packages

- `internal/task` - the `Task` domain type and the file-backed persistence primitives every
  slice builds on: `ReadTask`, `WriteTaskFile`, and `MutateTask` (lock → read → validate/mutate →
  write), plus shared helpers (`errors.go`, `labels.go`, `id.go`, `commit_timing.go`) and
  `GitClient` (`git.go`).
- `internal/cli/support` - CLI plumbing shared by every slice: building a `GitClient`/worktrees
  dir from root flags, parsing repeated `key=value` flags, rendering output (`--json` envelope or
  human-readable summary), and mapping errors to exit codes.

There is no `pkg/`; everything shared lives under `internal/`.

## Command Assembly Pattern

1. **Slice-level**: each slice's `cmd.go` exports a `Command()` func returning a `*cli.Command`.
2. **Stage-level**: `internal/cli/task/task.go` mounts every slice into the `task` command tree;
   `internal/cli/task/review/review.go` mounts `record`/`approve`/`reject` into `review`.
3. **Root-level**: `internal/cli/root.go`'s `rootCommand` mounts `task.Command()` under the
   `taskman` root with the root flags, and wires `initLogger` (`internal/cli/logger.go`) as its
   `Before` hook.

## CLI Conventions

- A slice's `cmd.go` `Action` parses flags/args into the request its domain function expects,
  calls the domain logic (a plain function taking `tasksDir` and other runtime deps directly,
  built on `internal/task`'s shared primitives), renders success via `internal/cli/support`
  (`support.PrintTask`/`support.PrintJSON`), and returns errors through `support.Fail`.
- Every flag has a `Usage` string written for someone who only has the compiled binary - no
  references to files or paths in this repo.
- Slices must not import `internal/cli` (it imports them, so that would cycle) - shared helpers
  go in `internal/cli/support`, which depends on `internal/task` and `urfave/cli` only.
- `next` is the one slice allowed to import sibling slices (`specify`, `implement`), since
  guiding a task means knowing what every stage's own commands would accept next; none of them
  import it back.

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
- CLI slices return errors; the command's `Action` hands them to `support.Fail`, which maps
  domain errors to exit codes (see `support.ExitCode`).

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

- Use `slog` package for logging with proper log level and attrs. The root `--log-level`/`--log-format` flags configure it (see `internal/cli/logger.go`).

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
