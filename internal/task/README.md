# `internal/task`

The pure functional core: the `Task` aggregate, its append-only event log, the compiled workflow,
and the two pure functions the frontends drive everything through - `Apply` and `Instruction`.

It performs no filesystem, Git, clock, logging, CLI, MCP, or rendering work. Every event carries
its own `At`, supplied by the caller; nothing here reads a clock.

## Model

- `TaskSummary` (`summary.go`) - the default (non-JSON) CLI summary of a task: a template over
  `{TaskID, Title, State, Instruction}`, rendered to text by `internal/task/view`.
- `Task` (`types.go`) - the aggregate. Its current state is derived: `Task.State()` returns the
  last entry of the append-only `StateHistory`, never a separate mutable field.
  `Task.Instruction()` (`instruction.go`) is the pure per-state projection of what should happen
  next - `{state, action}` with an action of `dispatch`, `run`, `wait`, or `done`.
- `TaskEvent` (`events.go`) - the closed set of facts that may progress a task. Each event type
  reports its own `Kind()`; the four review gates are distinct types per stage x reviewer.
- `NewTask` / `Task.Validate` (`types.go`) - construction and invariant checking. `Task.Clone`
  (`clone.go`) is the deep copy `Apply` uses so callers never observe partial mutation.

## Workflow

`workflow.go` is the single compiled source of truth (`definition` in `definition.go`): each state
(`states.go`) declares its instruction, whether it is terminal, and the events it accepts. A few
events are global (`EventEscalated`, `EventAbandoned`) and apply from any non-terminal state.
`EventUnblocked` is not global - it is `StateBlocked`'s only accepted event, and is the sole way
back out of that state short of `EventAbandoned`.

A transition has an optional `guard` (checked against the pre-reduce task), a `reduce` reducer
(what the event does to the task), and either a fixed destination `to`, ordered `routes` tried
against the post-reduce task, or a `resolve` function computing the destination directly -
`EventUnblocked`'s only use, since it resumes at whatever state precedes the current one in
`StateHistory` rather than one of a small fixed set of states.

Verification failures and implementation-review rejections are bounded by the gate's `AutoFix`
configuration (`autofix.go`): when a gate has auto-fix enabled and a `MaxRounds` cap, the reducer
records a `Blockage` once the recorded rounds exceed the gate's effective budget, and the
transition's `isBlocked` route sends the task to `StateBlocked` instead of another fix state.
Without a cap (disabled, or `MaxRounds <= 0`) the loop is unbounded. `EventUnblocked` can grant a
budget-exhausted gate more rounds: the grant is appended to that gate's `Unblocks`, never mutating
`MaxRounds` itself, so the effective budget (`effectiveMaxRounds`) is always the configured max
plus the sum of every grant on record - fully derived from history like everything else here.

`Apply(current, event)` validates `current`, looks the event up (global first, then state-local),
checks its guard, reduces a `Clone`, resolves the destination, appends the new state to
`StateHistory`, and validates the result. On any error it returns the zero `Task` alongside the
error, so a rejected event never yields a partially mutated task. Errors are defined in
`errors.go`.
