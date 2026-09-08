# Task

`internal/task` owns the `Task` domain type and its persistence: the
`tasks`, `tasks_labels`, `tasks_sessions`, and `task_review_results` tables.
It is a shared module-level package, not a vertical slice - the agent-facing
tools that expose task CRUD to a model (`task_create`, `task_get`,
`task_list`, `task_update`, `task_delete`) live under
`internal/agent/tools/task/<name>` and import this package for their actual
database work.

## Task

A `Task` tracks one unit of work: a definition and specification, a coarse
lifecycle `State` (`created`, `started`, `completed`, `cancelled`, `blocked`,
`failed`), six 1-5 planning `Level`s (importance, urgency, complexity,
effort, risk, autonomy), the model/reasoning effort it should run on, and,
once work has happened, a commit hash, branch, and failure reason.

`PipelineStep` is a plain string column, opaque to this package - it's the
autonomous pipeline's own progress marker (see `internal/pipeline.
PipelineStep`), stored here only because a task's pipeline progress is task
state, same as everything else on the row.

`CreateTask`, `GetTask`, `UpdateTask`, `DeleteTask` (soft delete), and
`ListTasks` (filtered, e.g. by state/label/level) are the CRUD surface.
Labels are replaced wholesale on create/update, not diffed.

## Sessions and review results

`LinkSession` records that a session was dispatched on a task's behalf
(`tasks_sessions`); `SessionIDsForTask` reads them back. Callers that
dispatch a session for a task (currently `internal/pipeline`) must link it
immediately after creating the session and before running it, so the link
survives even if the session's own turn crashes mid-run.

`InsertReviewResult` records one review attempt's outcome
(`task_review_results`): approved or not, plus structured findings
(`ReviewFinding{File, Summary}`). It's append-only - `attempt` auto-assigns
the next number for the task - so a crash after a review finishes never
loses what it found. `ReviewResultsForTask` reads every attempt back,
oldest first.
