# task_create

Creates a task in the `created` state and persists it in SQLite. The tool
accepts the task definition, specification, planning levels, labels, model,
and optional commit hash, and returns the generated task ID.

The tool requires a configured database through `tools.Deps.Store`.
