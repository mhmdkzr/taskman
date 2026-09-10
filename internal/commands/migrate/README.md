# migrate

`taskman migrate` validates and rewrites legacy schema-v1 task files into the
single-state schema used by the compiled functional workflow. It first converts
and validates every task, then writes each file atomically under its task lock.

Use `taskman migrate --dry-run` to show every inferred state without changing
files. Files already on the current schema are skipped. Ambiguous or invalid
legacy states stop the migration rather than being guessed.
