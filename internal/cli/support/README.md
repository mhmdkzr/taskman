# `internal/cli/support`

Plumbing shared by every taskman command slice in `internal/cli/task` and
`internal/cli/task/review`:

- `GitFrom`/`WorktreesDirFrom` - build an `internal/task.GitClient`/worktrees dir from the root
  `--git-dir`/`--worktrees-dir` flags.
- `RequireID` - read the task id positional argument.
- `SplitKV`/`ParseChecks`/`ParseFindings` - parse repeated `key=value` flags.
- `PrintTask`/`PrintJSON`/`CurrentStage` - render a `Task` as the full JSON envelope behind
  `--json`, or a short human-readable summary otherwise (via its own embedded `task_summary.md`).
- `Fail`/`ExitCode` - map an `internal/task` error to a `cli.ExitCoder` with the right exit code.

This package depends on `internal/task` and `urfave/cli` only - never on `internal/cli` or any of
its subpackages, so every slice under `task`/`task/review` (which `internal/cli` imports) can
depend on it without a cycle. This is also why it can't own `next`'s guidance-printing logic:
`next` is itself one of those slices, so `next/cmd.go` inlines that piece directly instead.
