# Plan: landing the taskman design

Execution plan for the design in `design.md`. This file tracks *how and in what order* to build
it from the current codebase - the design doc itself stays pure design, no sequencing.

## Already done

- GitHub integration removed (`internal/github` deleted, along with `internal/agent`,
  `internal/web`; no `internal/webhooks` ever existed in this repo to remove - an earlier design
  pass asserted it did, without checking; that claim is gone from `design.md` now, and this note
  exists so it doesn't quietly reappear in a future pass).
- Directory flags, no config file: `--git-dir`/`--tasks-dir`/`--worktrees-dir` plus
  `--log-level`/`--log-format`, all persistent root flags on the `taskman` command
  (`internal/cli/root.go`). Prompt templates ended up embedded in the binary instead of a
  `--prompts-dir` flag (see below) - `design.md` §1/§4 were updated to match.
- `internal/prompts`: every natural-language template taskman renders - not just the five
  judgment-dispatch prompts design.md originally named, but the dispatch wrapper and every
  `run`/`wait`/`done`/CLI-summary message too (13 templates total) - each embedded via
  `go:embed`, with its own typed params struct and `Render()` method. This was a deliberate
  scope widening from the original plan: embedding (not `os.ReadFile` from a `--prompts-dir`)
  and one struct per message shape, not just per judgment-dispatch prompt.
- `internal/task` rewritten from scratch for `.tasks/*.yaml` per design §3/§6: the new `Task`
  type (`types.go`), the file-backed `Repo` (`repo.go`), `GitClient` (`git.go`), id generation
  (`id.go`), label validation (`labels.go`), and every command in §6's table (`create.go`,
  `update.go`, `delete.go`, `workflow.go`, `next.go`). Full test coverage: `repo_test.go`,
  `id_test.go`, `git_test.go`, `workflow_test.go`, `next_test.go`.
- `internal/store`, `migrations/`, `pkg/migrate`, and the `modernc.org/sqlite` dependency
  dropped. Scope widened past the original plan once design.md settled on CLI-only (no HTTP,
  no MCP): the entire dead HTTP app framework went with it too -  `internal/app` (config,
  process, register, routes), `internal/mcp`, `internal/providers/opencode`, and
  `pkg/middleware`/`pkg/jsonresp`/`pkg/pagination` (all either only served that framework or,
  for `pkg/pagination`, had zero users already). `cmd/main/main.go` rewritten as the `taskman`
  CLI entrypoint (later trimmed to a two-line `main()` once `internal/cli` existed - see below).
  `config.yaml`, `.tasks/abc.yaml`, and `agents/review.yaml` (stale pre-design sketches) and
  `.env.example` (no env vars left to document) deleted.
- The taskman command layer and CLI (design §6/§7): CRUD, workflow commands, `Next`, and
  `urfave/cli` v3 wiring (`cmd/main/*.go`, later moved to `internal/cli`) - all landed together rather than as 5 separate
  passes, once the shape of `internal/task` was settled. `--json` plus the directory flags are
  persistent root flags, inherited by every subcommand. HTTP/MCP adapters: not built, per
  design §7's CLI-only scope.
- Manual end-to-end testing (a real git repo, real worktrees, real commits) walked the full
  happy path, the review-reject-recovery cycle, the two-round automated-review-rejection cap
  (blocked), escalate, abandon, delete, concurrent same-task mutations, and the dirty-working-
  tree precondition - see "Found during implementation" below for what that surfaced. Automated
  coverage for the same paths now lives in `internal/cli/cli_integration_test.go`
  (`TestCLIFullLifecycle` and friends) and `internal/task/*_test.go`.

## Found during implementation

Two correctness issues and one operational requirement, none anticipated in `design.md`, all
fixed and reflected there now:

- **The per-task `flock` was taken on the task file itself, not a separate lock file.**
  `write()`'s atomic rename replaces the task file's inode on every write; a lock survives only
  as long as the inode it was opened against, so a concurrent process's fresh `open()` onto the
  post-rename inode would acquire an independent, non-contending lock - silently defeating
  cross-process exclusion the moment a write ever succeeded. Caught by an automated concurrency
  test (30 concurrent `task update` calls losing writes), fixed by locking a separate, stable
  `<id>.yaml.lock` file instead. `design.md` §3's Concurrency bullet now describes this
  explicitly.
- **`.worktrees/` must be gitignored.** Discovered manually: after one `task create`, its
  worktree is an untracked directory, which fails the *next* `task create`'s clean-working-tree
  check. Not a code bug - an operational precondition the design never stated. Documented in the
  root `README.md`'s quickstart and baked into every test fixture (`newTestRepo` in both
  `internal/task` and `internal/cli` tests commits a `.gitignore` with `.worktrees/` up front).
- **`<id>.yaml.lock` files need to be gitignored too**, for the same reason - `.gitignore` now
  has a `.tasks/*.lock` entry.
- **`--log-level`/`--log-format` were declared as root flags but never actually wired to
  anything** - `pkg/logger.Init` was never called from `cmd/main`, so the flags parsed but did
  nothing. Found by re-checking test coverage against the full flag surface rather than just the
  happy-path commands. Fixed by adding a `Before` hook (`internal/cli/root.go`'s `initLogger`)
  that sets up `slog` from those flags before any command runs, and manually verified (both a
  successful run and an invalid `--log-level` erroring as expected).

## Post-implementation cleanup

- CLI code moved out of `cmd/main` into `internal/cli`, leaving `cmd/main/main.go` as a two-line
  `func main() { os.Exit(cli.Run()) }`. `internal/cli` is package `cli` - it can still import
  `github.com/urfave/cli/v3` under its default `cli` identifier from inside itself without
  collision, since a package never qualifies its own exported names.
- `pkg/logger` removed - it wrapped `slog.Init` for exactly one caller (`internal/cli`'s
  `initLogger`), so the ~15 lines were inlined directly rather than kept as a separate package.
- `pkg/testenv` removed - zero callers anywhere in the codebase once the old SQLite/HTTP stack
  was gone; nothing left to gate or load `.env` for.
- `internal/cli` split further: each `task <command>` moved to its own file under
  `internal/cli/task` (`list.go`, `create.go`, ...), with the review stage's three commands under
  `internal/cli/task/review`. `internal/cli/support` holds the plumbing both need (repo/git
  construction, output rendering, error mapping) - `internal/cli` itself imports `task` to mount
  its commands, so `task`/`task/review` can't import `internal/cli` back for shared helpers
  without a cycle; `support` depends on neither. Every command file has its own `_test.go`, plus
  a `testutil_test.go` per package for shared fixtures (`task/review`'s builds its test task via
  `internal/task` directly rather than `internal/cli/task`'s commands, for the same cycle reason).
  Every flag also got a real `Usage` string, written for someone who only has the compiled binary
  - no file or path references from this repo.
- `internal/cli/task` (and `internal/cli/task/review`) restructured again, from one file per
  command into one vertical slice package per command (`internal/cli/task/list`, `.../create`,
  etc., `.../review/record`, `.../review/approve`, `.../review/reject`) - each owning its own
  `cmd.go` (CLI wiring) and `<name>.go` (domain logic), following this repo's
  `internal/<module>/<feature>` convention. `internal/task`'s `Repo` type is gone: its
  lock/read/write logic became three plain functions (`ReadTask`, `WriteTaskFile`, `MutateTask`
  in `internal/task/store.go`) that every slice calls directly, and every workflow command that
  used to live in `internal/task` (`create.go`, `update.go`, `delete.go`, `workflow.go`,
  `next.go`) moved into its slice's own package instead. `internal/prompts` is gone the same
  way - each of its 13 templates moved to the one slice that uses it (`specify`, `implement`, and
  `create`'s own CLI summary each own their `prompt.md`+`prompt.go`), except `task_summary.md`
  (used by every slice's `--json`-off output, now embedded in `internal/cli/support`) and the 9
  templates behind `next`'s guidance logic, which all moved into the `next` slice together since
  nothing outside it used them. `next` is the one slice that imports other slices (`specify`,
  `implement`, for their `Prompt` structs) - safe now that `next` itself lives in
  `internal/cli/task/next` rather than `internal/task`, so it's on the same side of the
  `cli`→`cli/task`→`task` import direction as the slices it needs. Two pieces of `next`'s old
  logic (`HasCommitSince`, `NeedsFreshCommit`) stayed behind in `internal/task/commit_timing.go`
  specifically because `internal/cli/support.CurrentStage` needs `NeedsFreshCommit`, and
  `support` can't import `next` without a cycle.

## Resolved since first written

- Build-check attempts get their own audit list: `verifications[]` in the task schema, one entry
  per `task verify` call with a `checks: {name: ok|error}` map (not a single aggregate
  pass/fail), `output`, `created_at` - mirrors `reviews[]`.
- The `blocked` overlay is one task-level object (`{stage, reason, at}`), not a per-stage
  `needs_human` value - resolved during design, not left open.
- `task.State` drops `cancelled` - `failed` (via `task abandon`) already covers "abandoned before
  any work happened" without a second terminal state meaning almost the same thing.
- Commit timing: exactly when `verification.state` first reaches `done` (automated review
  approved), never at `merge`, and never for a task that ends up `blocked` instead. A
  review-reject-recovery cycle that clears produces its own new commit, never an amend.
- No config file at all - directories are CLI flags with defaults, resolved fresh on every
  invocation (no `config.yaml`, no `.env`, no interpolation).
- No session tracking of any kind - dropped entirely, not just moved out of taskman's own file
  format.
- `task commit` takes no `--message`/`--hash`: taskman reads the caller's already-made commit
  directly via `git log` instead of trusting reported text.
- Task ids are `<uuid-v7>_<slug>`, not short freeform slugs - no collision handling needed.
- Automated review (`reviews[]`) and human review (`human_reviews[]`) are explicitly two
  different logs with two different shapes - full structured findings for the former, a single
  flat text block for the latter - never to be conflated.
- HTTP and MCP are out of scope for this design entirely (not merely deferred past the CLI) -
  taskman is CLI-only.
- Prompt templates are embedded in the binary (`go:embed`), not read from a `--prompts-dir` at
  runtime - a deliberate implementation-time decision, reflected back into design.md.

## What's left

Nothing from the original plan. Possible future work, not currently scoped: HTTP/MCP adapters
(explicitly out of design's scope, would need a fresh design discussion first), a `loop` project
to actually drive taskman unattended (a separate repo per design §0), and more built-in prompt
variety if real usage shows the current five judgment-dispatch prompts too generic.
