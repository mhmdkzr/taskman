# taskman

A file-backed, stateless, one-shot CLI task server. No database, no GitHub integration, no HTTP
server, no MCP - just task files, prompt templates, and a CLI that validates and records the state
transitions reported to it. See `notes/design/design.md` for the full design.

## Quick Start

```bash
git init myproject && cd myproject
git commit --allow-empty -m "chore: init"
echo ".worktrees/" >> .gitignore && git add .gitignore && git commit -m "chore: ignore worktrees"

go run github.com/mhmdkzr/taskman/cmd/main task create --definition "Fix doc drift in internal/task"
```

`.worktrees/` **must** be gitignored before the first `task create` - taskman's `task create`
refuses to run against a dirty working tree (§5), and the worktree it creates is itself untracked,
which would make every task after the first fail that check. `.tasks/*.yaml` is meant to be
tracked and committed, not ignored.

From there, `task next <id>` tells you (or whatever agent you're driving) what to do next at every
step - see `notes/design/design.md` §6/§7 for the full command reference and state machine.

## Project Layout

| Path | Purpose |
| --- | --- |
| `cmd/main` | The `taskman` binary's minimal entrypoint |
| `internal/cli` | The `taskman` CLI's root command and flags |
| `internal/cli/support` | Plumbing shared by every command slice (git construction, output, errors) |
| `internal/cli/task` | Every `task <command>`, one vertical slice package each |
| `internal/cli/task/review` | The review stage's `record`/`approve`/`reject` slices |
| `internal/task` | The `Task` domain type and shared file-backed persistence primitives |
| `notes/design/` | The design (`design.md`) and its execution plan (`plan.md`) |

## Commands

```bash
make lint
make fmt
make test
```

## Dependencies

- Go 1.27+
- `git` on `PATH` (taskman shells out to it for worktrees and reading commits)
