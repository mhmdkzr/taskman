# Plan: landing the taskman design

Execution plan for the design in `design.md`. This file tracks *how and in what order* to build
it from the current codebase - the design doc itself stays pure design, no sequencing.

## Already done

- GitHub integration removed (`internal/github` deleted, along with `internal/agent`,
  `internal/web`; no `internal/webhooks` ever existed in this repo to remove - an earlier design
  pass asserted it did, without checking; that claim is gone from `design.md` now, and this note
  exists so it doesn't quietly reappear in a future pass).

## Remaining work

1. **`config.yaml` loading.** Plain `yaml.Unmarshal` into `Config` under the `taskman:` key plus
   `Config.Validate()` - no `.env`, no `${VAR}` interpolation, no `server:`/`logger:` sections
   (logging becomes a CLI flag). Just `Config.Dirs`. Today's `internal/app/config` is still the
   old env-var-driven `Server`/`Logger`/`Database`/`Provider` shape - this step replaces it, not
   extends it.
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
   is its own process, so an in-memory lock provides no protection). Includes `sessions:`/
   `reviews[].session` as opaque id strings, `human_reviews[]`, the `blocked` overlay, and
   `verifications[]` (one entry per `task verify` call, per-check `ok`/`error` status) from day
   one - these are all part of the same `Task` type, not separate follow-on work.
4. **Drop `internal/store`, `migrations/`, `pkg/migrate`, and the `modernc.org/sqlite`
   dependency.** Not just deleting the packages - `internal/app/app.go` (`Deps.Store`),
   `internal/app/process/start.go` (`store.Open`, `runMigrations`), and
   `internal/app/config/sqlite.go` all still wire up SQLite today and need updating in the same
   pass, or step 4 leaves the build broken.
5. **The taskman command layer and CLI (design §6/§8).** The largest single piece of work;
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
   4. CLI wiring (`urfave/cli` v3) over all of the above - one `cli.Command` per row in §8's
      command table, `--json` as a persistent root flag.
   5. HTTP and MCP adapters - deferred, not required for the CLI to be usable; land only once
      5.1-5.4 are solid.

Each step keeps its own README updated and lands with the smallest sensible `go build`/`go vet`/
test scope, as separate, package-scoped commits.

## Resolved since first written

- Build-check attempts get their own audit list: `verifications[]` in the task schema, one entry
  per `task verify` call with a `checks: {name: ok|error}` map (not a single aggregate
  pass/fail), `output`, `session`, `created_at` - mirrors `reviews[]`.
- The `blocked` overlay is one task-level object (`{stage, reason, at}`), not a per-stage
  `needs_human` value - resolved during design, not left open.
- `task.State` drops `cancelled` - `failed` (via `task abandon`) already covers "abandoned before
  any work happened" without a second terminal state meaning almost the same thing.
- Commit timing: exactly when `verification.state` first reaches `done` (automated review
  approved), never at `merge`, and never for a task that ends up `blocked` instead. A
  review-reject-recovery cycle that clears produces its own new commit, never an amend.
