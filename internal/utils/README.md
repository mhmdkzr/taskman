# `internal/utils`

CLI-only plumbing shared by more than one command slice's `cmd.go`. It has no dependency on
`internal/commands`, so any slice can import it without a cycle.

- `IDFrom` - parse the `--id` flag as a task id.
- `SplitKV` / `ParseFindings` - parse repeated `key=value` flags and review findings.
- `ReviewFlags(prefix)` / `ReviewConfigurationFrom(cmd, prefix)` - the shared `--agent-review` /
  `--human-review` flag set (every flag name prefixed by `prefix`) and its mapping to
  `task.ReviewConfiguration`. `specified` calls this twice - once with `""` for the
  specification-review gate, once with `"impl-"` for the implementation-review gate - now that both
  are declared at specify time.
- `RequireFlagsIf` - like `RequireFlags`, but only enforced when a condition holds; used by
  `specified` for `--worktree`/`--branch`, which are required together only when `--use-worktree`
  is set.
- `Fail` - wrap domain errors into `cli.ExitCoder` errors (1 for domain failures, 2 for malformed
  input); the process exit code is mapped by `internal/cli`.
- `SchemaFor` - JSON Schema inference for a slice's `Request`/output type, with `uuid.UUID`
  registered as a string so an MCP tool's schema matches its wire format.

Output rendering lives in `internal/task/view` (`view.PrintTask`/`view.PrintJSON`/
`view.PrintMarkdown`); the default task summary template lives in `internal/task/summary.go`.
Anything used by exactly one slice lives in that slice.
