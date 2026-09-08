# `internal/cli`

The `taskman` CLI's entrypoint. `Run()` builds the root `urfave/cli` v3 command
(`rootCommand`, `root.go`) and runs it once; there is no persistent daemon.

```
taskman [--git-dir <dir>] [--tasks-dir <dir>] [--worktrees-dir <dir>] [--json]
        [--log-level <level>] [--log-format <format>]
        task <command> [args...]
```

`root.go`'s `Before` hook (`initLogger`) sets up `slog` from `--log-level`/`--log-format` before
any command runs. Everything under `task <command>` - list/get/create/update/specify/implement/
verify/review/commit/escalate/merge/abandon/next/delete - is wired up in `internal/cli/task`
(and its `review` subpackage), not here; `rootCommand` just mounts `task.Command()`.

## Subpackages

| Package | Purpose |
| --- | --- |
| `support` | Shared plumbing: building a `Repo`/`GitClient` from root flags, parsing repeated flags, rendering output, mapping errors to exit codes |
| `task` | Every `taskman task <command>`, one file per command |
| `task/review` | The review stage's `record`/`approve`/`reject` commands |

`support` exists specifically to avoid a cycle: `cli` imports `task` (to mount its commands),
so `task` and `task/review` can't import `cli` back for shared helpers - they import `support`
instead, which depends on neither.
