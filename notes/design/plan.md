# Plan: landing the taskman design

Execution plan for the design in `design.md`. This file tracks *how and in what order* to build
it from the current codebase - the design doc itself stays pure design, no sequencing.

1. **`config.yaml` loading.** Plain `yaml.Unmarshal` into `Config` under the `taskman:` key plus
   `Config.Validate()` - no `.env`, no `${VAR}` interpolation, no `server:`/`logger:` sections
   (logging becomes a CLI flag). Just `Config.Dirs`.
2. **`prompts/*.md`.** Smallest content migration - plain files, no metadata to validate beyond
   existence. Ship the built-in prompts (`specify`, `implement`, `fix`, `review`, `commit`)
   checked into the repo.
3. **`.tasks/*.yaml` (the task repo).** Read/write/list per the design's repo implementation
   (§3): `os.ReadFile`/`yaml.Unmarshal` for reads, write-to-temp-then-rename for writes, an
   OS-level `flock` per task file for cross-process safety (taskman is a one-shot CLI - every
   invocation is its own process, so an in-memory lock provides no protection). Includes
   `sessions:`/`reviews[].session` as opaque id strings from day one.
4. **Drop `internal/store`/migrations/`modernc.org/sqlite` outright** - once (1)-(3) land,
   nothing references them.
5. **Remove GitHub integration** entirely (`internal/github`, `internal/webhooks`, issue-driven
   pipeline triggering) per §5.
6. **The taskman interface (§8)**: the command layer (`internal/task`'s one-function-per-command
   core, including `Create`'s clean-working-tree check + `git worktree add`, `Commit`, and
   `Update`) plus the CLI (`urfave/cli` v3) first; HTTP and MCP adapters after.

Each step keeps its own README updated and lands with the smallest sensible `go build`/`go vet`/
test scope, as separate, package-scoped commits.

## Resolved since first written

- Build-check attempts get their own audit list: `verifications[]` in the task schema, one entry
  per `task verify` call with a `checks: {name: ok|error}` map (not a single aggregate
  pass/fail), `output`, `session`, `created_at` - mirrors `reviews[]`.
