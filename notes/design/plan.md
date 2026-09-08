# Plan: landing the taskman design

Execution plan for the design in `design.md`. This file tracks *how and in what order* to build
it from the current codebase - the design doc itself stays pure design, no sequencing.

## Already done

- GitHub integration removed (`internal/github` deleted, along with `internal/agent`,
  `internal/web`; no `internal/webhooks` ever existed in this repo to remove - an earlier design
  pass asserted it did, without checking; that claim is gone from `design.md` now, and this note
  exists so it doesn't quietly reappear in a future pass).

## Remaining work

1. **Directory flags, no config file.** `--git-dir` (default `.`), `--tasks-dir` (default
   `.tasks`), `--worktrees-dir` (default `.worktrees`), `--prompts-dir` (default `prompts`),
   plus `--log-level`/`--log-format` - all resolved from CLI flags at invocation, nothing loaded
   from disk before a command runs. Today's `internal/app/config` is still the old
   env-var-driven `Server`/`Logger`/`Database`/`Provider` shape, loaded from `config.yaml` - this
   step removes that whole config-file-loading path, not adapts it. No `Config.Validate()`
   either: there's no file to have gotten wrong.
2. **`prompts/*.md`.** Smallest content migration - plain files, no metadata to validate beyond
   existence. Ship the built-in prompts (`specify`, `implement`, `fix`, `review`, `commit`)
   checked into the repo.
3. **Rewrite `internal/task` for `.tasks/*.yaml`.** Not new construction - `internal/task` is
   today a complete, working SQLite-backed package (`types.go`'s `Task` struct uses `uuid.UUID`
   IDs, `Importance`/`Urgency`/`Complexity`/`Effort`/`Risk`/`Autonomy` `Level` fields, a
   `ReasoningEffort` string, a `PipelineStep` field tied to the now-deleted pipeline package, and
   a `TaskStateCancelled` value design.md explicitly drops) - that whole type and its
   SQL-backed repo (`repo.go`) get replaced, not added to. New repo per design §3:
   `os.ReadFile`/`yaml.Unmarshal` for reads, write-to-temp-then-rename for writes, an OS-level
   `flock` per task file for cross-process safety (taskman is a one-shot CLI - every invocation
   is its own process, so an in-memory lock provides no protection). `id` is `<uuid-v7>_<slug>`
   (§3) - a UUID v7 generator (Go's stdlib `uuid` package, per `CLAUDE.md`) plus a title-to-slug
   helper. Includes `human_reviews[]` (flat `{approved, comment, at}` entries, no nested findings
   - distinct from `reviews[]`'s structured, full-text `findings[].detail`), the `blocked`
   overlay, and `verifications[]` (one entry per `task verify` call, per-check `ok`/`error`
   status) from day one - these are all part of the same `Task` type, not separate follow-on
   work. No `sessions`/session-id field anywhere - dropped from the design entirely.
   `Commit`'s implementation needs a small `Git` helper (`git log` in a worktree, parsing hash +
   message + a leading conventional-commit `type:` prefix) - taskman reads the real commit back
   rather than trusting caller-reported text (design §6, "When the commit happens").
4. **Drop `internal/store`, `migrations/`, `pkg/migrate`, and the `modernc.org/sqlite`
   dependency.** Not just deleting the packages - `internal/app/app.go` (`Deps.Store`),
   `internal/app/process/start.go` (`store.Open`, `runMigrations`), and
   `internal/app/config/sqlite.go` all still wire up SQLite today and need updating in the same
   pass, or step 4 leaves the build broken.
5. **The taskman command layer and CLI (design §6/§7).** The largest single piece of work;
   broken down here specifically because it isn't one commit:
   1. Core task CRUD: `Create` (including the clean-working-tree check and `git worktree add`
      exception, §5), `Update`, `Delete`, plus rewiring existing `Get`/`List` onto the new repo
      from step 3.
   2. Workflow commands, in the order a task actually moves through them: `Specify` → `Implement`
      → `Verify` → `RecordReview` → `Commit` → `Escalate` → `ApproveReview`/`RejectReview` →
      `Merge` → `Abandon`. Each is small and independently testable against §6's precondition/
      effect table.
   3. `Next` - deliberately last among the Go functions, since it reads the state every other
      command in 5.2 produces (`dispatch`/`run`/`wait`/`done`, the `message`/`report_with`
      contract) and has nothing to inspect until they exist.
   4. CLI wiring (`urfave/cli` v3) over all of the above - one `cli.Command` per row in §7's
      command table, `--json` plus the directory flags from step 1 as persistent root flags.
      This is the whole interface - no HTTP or MCP adapter follows; design §7 puts both out of
      scope entirely, not deferred.

Each step keeps its own README updated and lands with the smallest sensible `go build`/`go vet`/
test scope, as separate, package-scoped commits.

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
