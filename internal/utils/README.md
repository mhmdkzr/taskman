# `internal/utils`

CLI-only plumbing shared by every command slice's `cmd.go`. It has no dependency on
`internal/commands`, so any slice can import it without a cycle.

- `StoreFrom` / `GitFrom` / `IDFrom` - build a `*store.Store` and `*git.Client` from the root
  flags, and parse `--id`.
- `SplitKV` / `ParseFindings` / `ParseCheckResult` - parse repeated `key=value` flags, review
  findings, and `ok`/`error` check values.
- `ReviewFlags` / `ReviewConfigurationFrom` - the shared `--agent-review` / `--human-review` flag
  set and its mapping to `task.ReviewConfiguration`, used by `specified` and `implemented`.
- `PrintTask` / `PrintJSON` / `PrintMarkdown` - the three renderings: default summary, `--json`
  envelope, `--md` document.
- `Fail` / `ExitCode` - wrap domain errors and map them to process exit codes (1 for domain
  failures, 2 for malformed input).
- `SchemaFor` - JSON Schema inference for a slice's `Request`/output type, with `uuid.UUID`
  registered as a string so an MCP tool's schema matches its wire format.
