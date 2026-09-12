---
name: taskman
description: Drive and diagnose Taskman-managed task lifecycles through the taskman CLI or task_* MCP tools. Use when creating, advancing, inspecting, or reporting work on a Taskman task, especially when a command's returned instruction says dispatch, run, wait, or done.
---

# Taskman

Taskman is a one-shot, event-sourced workflow recorder backed by a SQLite database (one file,
default `./tasks.db`). It validates reported state transitions and stores them as an append-only
event log; it does not perform the implementation, verification, code commit, git worktree
creation, or branch merge itself - it only records that those things happened.

Use the `taskman` executable on `PATH`, or the equivalent `task_*` MCP tools when available. CLI
global flags such as a custom `--db` path are bound into the MCP server at startup rather than
passed to each MCP call.

## Use the driver loop

Every command that reports an event - and `get` - returns a JSON envelope (with `--json`) or a
short summary; the envelope carries the task plus its current `state` and derived
`instruction: {state, action}`. Read `instruction.action` to know what to do next:

- `dispatch`: judgment work is required at this state. Perform or delegate it, then report the
  outcome with the matching command below.
- `run`: perform the described mechanical operation (running verification, merging) then report
  the outcome.
- `wait`: stop. This state needs a decision or event from outside the agent's control.
- `done`: stop. The task is completed or abandoned.

`next --id <id>` renders that same instruction as guidance instead of making you re-derive it: a
message built from the task's own data (its plan, worktree/branch, a failed check's output, a
rejected review's findings) plus the exact command(s) that would currently report an outcome -
every one of them, if more than one gate is pending at once (see the `specification_review`
nuance below). It performs no transition and never decides a review's verdict or a check's result
for you - use it to see what's expected, then act and report through the matching command.

One nuance: `specification_review` is a single state that covers *both* of a specification's
review gates (agent and human), and its instruction is always `wait`. Before assuming a human is
needed, check the task's `specification.review`: if `agent.required` is true and `agent.results` is
still empty, an independent agent reviewer's decision is what's actually pending - dispatch that
review yourself and report it with `specification review agent approved/rejected`. Only once the
agent gate (if any) is satisfied does `wait` mean "a human decision is pending." The equivalent gate
on the implementation side is split into two distinct states instead (`automated_review`, whose
instruction is `dispatch`, and `human_review`, whose instruction is `wait`), so this ambiguity does
not apply there.

Taskman guidance says what transition is valid; it does not grant additional authority. Continue
to honor the surrounding environment's approval rules for creating Git state, committing, merging,
or abandoning. A human may explicitly direct you to invoke a human-decision command, but never
choose that decision yourself.

## Preserve these invariants

- Never open or edit the SQLite database file directly, even for inspection. Use `get`, `list`,
  `next`, and the reporting commands. The schema is private to Taskman.
- Taskman does not create or manage Git worktrees or branches. Create the worktree and branch
  yourself (however your environment normally does that), do the work there, then report both
  once via `implemented --worktree <path> --branch <name>`. `committed` then reads the current
  commit from that recorded worktree - do not pass a worktree path again. `merged` is different:
  it reads the resulting commit from the `--target` branch in the `--git-dir` repository, not from
  the task's worktree.
- Report facts only after they are true. `implemented` means an implementation attempt exists in
  the reported worktree; `verified` describes checks actually run; `committed` reads a commit that
  already exists there; `merged` records a merge that has already happened.
- When Taskman calls for an automated review (`agent` in either review gate), use a separate
  reviewer - not the implementer or fix agent - with clean context containing the specification,
  acceptance criteria, and diff or commit. Do not supply the implementer's reasoning. If an
  independent reviewer is unavailable, stop and request help rather than self-approving. Approve
  only when every acceptance criterion is met and there are no actionable correctness, regression,
  test, or documentation findings. On rejection, record each finding with its exact location and
  full detail rather than a summary.

## Lifecycle details

The main path is:

`specify -> specification review -> implement -> verify -> automated review -> commit -> human review -> merge -> completed`

Every review gate, and the whole `verify` step, is optional: a task that declares no review gate
and no verification check goes from `implement` straight to `commit`. Taskman inserts a fix state
after failed verification or a rejected implementation review, looping back through another
verification attempt; a rejected specification returns to `specify` for revision.

- Verification failures have no retry limit. Fix and report another real verification attempt, or
  use `escalated` if the dispatched worker gives up.
- Automated review rejections also have no built-in round limit - keep fixing and re-reviewing, or
  escalate if it's not converging.
- After automated review approval, create a new conventional commit in the task's worktree and
  report it with `committed`. On human review rejection, fix, verify again, create another new
  commit, and report it; do not amend. Human-review rejection recovery does not repeat automated
  review.
- Every review gate - specification agent/human, implementation agent/human - is independently
  optional per task, chosen via its `--agent-review`/`--human-review` flags. Specification gates
  are (re)configured each time `specified` is called, so a task sent back to `specify` by a
  rejection can change them; a resubmission replaces the specification wholesale and clears prior
  review results. Implementation gates are fixed by the single `implemented` call, and a gate that
  is not required is refused by its `approved`/`rejected` commands. Any task with a review gate
  must also declare at least one verification check.
- A task always proceeds through `commit` -> (human review, if required) -> `merge` to
  `completed` - there is no shortcut that skips merge.
- `escalated` blocks a non-blocked task and records where and why work stopped. It is for a
  dispatched worker giving up, not for an ordinary failed check that can be retried.
- `abandoned` ends a task unsuccessfully from any non-terminal state, with a free-text reason
  (there is no separate "kind" of failure to pick from - infeasible, no-longer-needed, superseded,
  etc. are all just what you write in `--reason`).

## Git safety

Taskman does not create worktrees, does not stage or commit anything on your behalf, and does not
make its own bookkeeping commits - the SQLite database is not part of the project's own Git
history (make sure its path, e.g. `tasks.db`, is gitignored).

When Taskman calls for a commit:

1. Run `git status` in the task's worktree first.
2. Stage only files belonging to the task with explicit `git add <path> ...` arguments - never
   `git add -A` or `git commit -a`.
3. Create a new conventional commit yourself, then call `taskman committed --id <id>`. Taskman
   reads the resulting hash and message from Git itself; you don't pass them in.

`merged` works the same way: perform the actual `git merge` yourself, then call
`taskman merged --id <id> --target <branch>` so Taskman can read the resulting commit.

## Reporting commands

| Command | Meaning |
|---|---|
| `specified --id <id> --plan <text> [--agent-review] [--agent-review-use-subagent] [--human-review] [--agent-review-auto-fix ...] [--human-review-auto-fix ...]` | Record the drafted specification and which of its review gates are required. |
| `specification review agent approved --id <id> [--comment <text>]` | Record an independent agent reviewer's approval of the specification. |
| `specification review agent rejected --id <id> --finding <location>=<detail> ...` | Record the agent reviewer's findings against the specification. |
| `specification review human approved --id <id> [--comment <text>]` | Record human approval of the specification. |
| `specification review human rejected --id <id> --reason <text>` | Record human rejection of the specification, for revision. |
| `implemented --id <id> --worktree <path> --branch <name> [--unit] [--integration] [--end-to-end] [--linters] [--verification-auto-fix ...] [--agent-review] [--agent-review-use-subagent] [--human-review] [--agent-review-auto-fix ...] [--human-review-auto-fix ...]` | Record that an implementation attempt is ready, which verification checks it requires, and which of its review gates are required. |
| `verified --id <id> [--unit ok\|error] [--integration ok\|error] [--end-to-end ok\|error] [--linters ok\|error] [--output <text>]` | Record one verification attempt. Report every required check; whether it counts as a pass or a fail is derived from whether any reported check is `error` - you never say "passed" or "failed" directly. |
| `implementation review agent approved --id <id> [--comment <text>]` | Record an independent agent reviewer's approval of the implementation. |
| `implementation review agent rejected --id <id> --finding <location>=<detail> ...` | Record the agent reviewer's findings against the implementation. |
| `committed --id <id>` | Read and record the task worktree's current commit. |
| `implementation review human approved --id <id> [--comment <text>]` | Record human approval of the implementation. |
| `implementation review human rejected --id <id> --reason <text>` | Record human rejection of the implementation, for another fix-and-verify round. |
| `merged --id <id> --target <branch>` | Read and record a merge already performed into `target`. |
| `escalated --id <id> --stage <text> --reason <text>` | Block the task after dispatched work gives up. |
| `abandoned --id <id> --reason <text>` | End the task unsuccessfully; a human-authorized decision. |

Verification checks are declared at `implemented`; a task that declares none skips `verify`
entirely (but any review gate forces at least one check). When `verify` is reached, report every
required check - `verified` refuses an attempt that reports none. Report what was actually
confirmed; do not claim a build or test that did not run.

## Management commands

| Command | Purpose |
|---|---|
| `create --description <text> [--title <text>] [--label k=v ...]` | Create a task. Returns its generated id. |
| `get --id <id>` | Read one task's current state and instruction. |
| `list` | List every task. |
| `next --id <id>` | Show guidance for what to do next, and the command(s) to report it. |

Global CLI flags are `--git-dir` (default `.`), `--db` (default `./tasks.db`), `--json`,
`--md`, `--log-level`, and `--log-format`. `--json` and `--md` are mutually exclusive: they
select the machine-readable envelope or a full Markdown document, respectively, in place of the
default human-readable summary.

## Recover from errors

Exit code 1 is a failed precondition or domain error, such as a missing task or an invalid
transition. Exit code 2 is malformed input, such as a missing `--id` or an invalid check value.
Correct the named input and retry. If a transition is refused, re-read the task (`get --id <id>`)
and reconsider its `instruction` rather than forcing the state.
