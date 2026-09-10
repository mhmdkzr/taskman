# `internal/taskstore`

The imperative YAML persistence adapter for the pure `internal/task` model.
`Update` holds a stable per-task lock, reads and validates the current value,
calls a value-returning update function, and atomically replaces the YAML file.
A failed update is never written.
