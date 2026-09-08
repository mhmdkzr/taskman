# `internal/cli/support`

Plumbing shared by every taskman command in `internal/cli/task` and `internal/cli/task/review`:

- `RepoFrom`/`GitFrom`/`WorktreesDirFrom` - build an `internal/task.Repo`/`GitClient` from the
  root `--tasks-dir`/`--git-dir`/`--worktrees-dir` flags.
- `RequireID` - read the task id positional argument.
- `SplitKV`/`ParseChecks`/`ParseFindings` - parse repeated `key=value` flags.
- `PrintTask`/`PrintGuidance`/`PrintJSON`/`CreateSummary`/`CurrentStage` - render a `Task` or
  `Guidance` as the full JSON envelope behind `--json`, or a short human-readable summary
  otherwise.
- `Fail`/`ExitCode` - map an `internal/task` error to a `cli.ExitCoder` with the right exit code.

This package depends on `internal/task`, `internal/prompts`, and `urfave/cli` only - never on
`internal/cli` or its subpackages, so `task`/`task/review` (which `internal/cli` imports) can
depend on it without a cycle.
