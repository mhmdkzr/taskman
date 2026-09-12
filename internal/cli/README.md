# `internal/cli`

Assembles the `taskman` root command and runs exactly one command per process - there is no
daemon.

`rootCommand` (`cli.go`) declares the global flags (`--git-dir`, `--db`, `--json`, `--md`,
`--log-level`, `--log-format`), wires `initLogger` as its `Before` hook, rejects `--json` together
with `--md`, and mounts every command slice directly under the root. `Run` parses `os.Args`,
executes, and maps the returned error to an exit code via `utils.ExitCode`.

The two review-stage grouping commands (`specification`, `implementation`) are mounted here too,
and are the only nesting under the root.
