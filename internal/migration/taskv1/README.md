# `internal/migration/taskv1`

The isolated reader for legacy task files. It converts the old coarse state and
per-stage status matrix into the single authoritative workflow state, preserving task data
and histories. Migration validates all files before writing any, locks each
file during replacement, and refuses ambiguous or inconsistent states.

Normal runtime packages do not import this package; only the `migrate` command
does. It can be removed when schema-v1 migration is no longer supported.
