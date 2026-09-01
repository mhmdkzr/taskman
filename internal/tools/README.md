# `tools`

The tool set an agent can call.

## What it provides

- `codebase` — `read`/`edit`/`glob`/`grep` plus `go_build`/`go_test`, bound to one `codebase.Repository` (one task's worktree) at construction time. `Tools` is the executor's read+write set; `ReadOnlyTools` (no `edit`) is the reviewer's set — enforced structurally, not by prompting. gofmt/goimports/go vet/staticcheck/golangci-lint are deliberately not tools: they're deterministic and run automatically as pipeline steps instead (see `internal/codebase.Repository.Format`/`Lint`).
- `task` — `task_create`/`task_search`/`task_get`/`task_edit` over the SQLite task backlog. Lifecycle transitions (start/complete/review) are intentionally not tools; those stay orchestrator-driven.
- `telegram` — send messages to and read the configured Telegram channel (`telegram_send` / `telegram_read`), gated on credentials.
- `spawn` — `spawn_subagent` / `subagent_result` for nested sub-agents, backed by a process-wide `Runner`.
- `schedule` — the `agent.run` wire contract (`RunRequest`, `RunSubject`, `MeaningfulOverride`) plus the `Schedule` / `ScheduleRecurring` helpers. The scheduling *tools* are intentionally not included; the durable scheduler runtime lives in `internal/scheduler`.

`tools.Tools(repo, tg)` assembles codebase + telegram. The task and spawn tools are appended by `agent.DefaultTools`.