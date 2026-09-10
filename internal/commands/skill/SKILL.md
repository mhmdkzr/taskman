---
name: taskman
description: Drive and diagnose Taskman-managed task lifecycles through the taskman CLI or task_* MCP tools. Use when creating, advancing, inspecting, or reporting work on a Taskman task, especially when task_next returns dispatch, run, wait, done, or report_with guidance.
---

# Taskman

Taskman is a one-shot workflow recorder backed by `.tasks/*.yaml`. It validates reported state
transitions; it does not perform the implementation, verification, code commit, or branch merge.
It does create a task's Git worktree/branch and automatically commits the task file when the task
becomes terminal.

Use the `taskman` executable on `PATH`, or the equivalent `task_*` MCP tools when they are
available. CLI global flags such as a custom `--tasks-dir` are bound into the MCP server at startup
rather than passed to each MCP call.

## Use the driver loop

Start or resume every task with `taskman next <id>` (MCP: `task_next`). Follow the returned
`action` and `message`, report the result using `report_with`, then call `next` again:

- `dispatch`: judgment work is required. Perform or delegate exactly the work in `message`, in the
  named worktree and branch, then report the outcome.
- `run`: perform the mechanical operation described in `message`, then report the outcome.
- `wait`: stop. A human decision or intervention is required.
- `done`: stop. The task is completed or abandoned.

`report_with` omits the executable name. At the CLI, prefix it with `taskman`; with MCP, call the
corresponding `task_*` tool. Treat the message as current task context: it already includes the
definition, specification, acceptance criteria, references, worktree, branch, and prior failure
details needed for that step.

Taskman guidance says what transition is valid; it does not grant additional authority. Continue
to honor the surrounding environment's approval rules for creating Git state, committing, merging,
abandoning, deleting, or pruning. A human may explicitly direct you to invoke a human-decision
command, but never choose that decision yourself.

## Preserve these invariants

- Never read or edit `.tasks/*.yaml` directly, even for inspection. Use `next`, `get`, `list`,
  `update`, and the reporting commands. The persisted schema is private to Taskman.
- Work only in the `git.worktree` on the `git.branch` reported by Taskman. Do not infer the path
  from the task id. A normal task uses an isolated worktree; a task created with `--trunk` uses the
  repository checkout and current branch.
- Do not treat `update --trunk` as a checkout migration. It changes the task's workflow mode flag
  but does not move the recorded worktree or branch. Change it only when the existing Git location
  is already appropriate.
- Do not create or update a task with `auto_approve` unless the human explicitly requested removal
  of the human review gate. It is a workflow policy choice, not an agent convenience.
- Report facts only after they are true. `implemented` means an implementation attempt exists;
  `verified` describes checks actually run; `committed` reads an existing commit; `merged` records an
  already completed merge.
- When Taskman dispatches automated review, use a separate reviewer—not the implementer or fix
  agent—with clean context containing the specification, acceptance criteria, and diff or commit.
  Do not supply the implementer's reasoning. If an independent reviewer is unavailable, stop and
  request help rather than self-approving. Approve only when every acceptance criterion is met and
  there are no actionable correctness, regression, test, or documentation findings. On rejection,
  record each finding with its exact file and full detail rather than a summary.

## Lifecycle details

The workflow is:

`specify -> implement -> verify -> automated review -> commit -> human review -> merge`

Taskman may insert a fix state after failed verification or rejected review.

- Verification failures have no numeric retry limit. Fix and report another real verification
  attempt, or use `escalated` if the dispatched worker gives up.
- Automated review allows at most two rejected rounds. A rejection routes through fix and
  verification; the second rejection blocks the task automatically.
- After automated approval, create a new conventional commit in the task worktree and report it
  with `committed`. On human rejection, fix and verify again, create another new commit, and report it;
  do not amend. Human-rejection recovery does not repeat automated review.
- Human review is a hard gate unless the task has `auto_approve`. Invoke `review approved` or
  `review rejected` only after a human explicitly supplies that decision.
- A trunk task completes when review completes because it needs no merge. A non-trunk task proceeds
  to merge after review.
- `escalated` blocks a non-blocked task and records where and why work stopped. It is for a
  dispatched worker giving up, not for an ordinary failed check that can be retried.

## Git safety

Before creating a task, the repository must be clean. The `create` command creates an isolated
worktree and branch unless `--trunk` is supplied.

When Taskman asks for a code commit:

1. Run `git status` in the reported worktree.
2. Stage only files belonging to the task with explicit `git add <path> ...` arguments.
3. Never use `git add -A` or `git commit -a`. In trunk mode, unrelated changes and the task's own
   changing YAML may share the checkout.
4. Create a new conventional commit, then call `taskman committed <id>`. Taskman reads its hash and
   message from Git; `--commit` selects a commit-ish other than `HEAD`.

When a task becomes terminal, Taskman creates a separate bookkeeping commit for its own task file.
This occurs during `merged`, `abandoned`, trunk `review approved`, or `committed` for a trunk task with
auto-approval. Account for this Git side effect before invoking those commands.

## Reporting commands

Use `next` rather than choosing a transition from this table; the table explains the inputs that a
returned `report_with` may require.

| Command | Meaning |
|---|---|
| `specified <id> --result <text> --done-when <text>` | Record the drafted specification and acceptance criteria. |
| `implemented <id>` | Record that an implementation attempt is ready for verification. |
| `verified <id> --check <name>=<ok\|error> ... [--output <text>]` | Record one verification attempt. Include every check actually run; all must be `ok` to pass. |
| `review recorded <id> --approved <bool> [--finding <file>=<detail> ...]` | Record an independent automated review. Preserve full finding details. |
| `committed <id> [--commit <commit-ish>]` | Read and record a commit that already exists. |
| `escalated <id> --stage <stage> --reason <text>` | Block the task after dispatched work gives up. Valid stages: definition, specification, implementation, verification, review, merge. |
| `merged <id> [--commit <hash>]` | Record a merge already performed; use the override for the resulting merge hash when needed. |

`verified` requires at least one check even for documentation-only or no-op work. Name the check for
what was actually confirmed, such as `review=ok` for a careful read-through; do not claim a build or
test that did not run.

## Management commands

| Command | Purpose |
|---|---|
| `create --definition <text> [--title <text>] [--id <id>] [--label k=v ...] [--reference <ref> ...] [--specification <text> --done-when <text>] [--trunk] [--auto-approve]` | Create a task. Specification and done-when must be supplied together and skip the specify state. |
| `get <id>` | Read one task through the supported interface. |
| `list [--state <state> ...] [--label k=v ...] [--limit <n>] [--offset <n>]` | List and filter tasks. The default limit is 50; `0` is unlimited. JSON output contains `tasks`, `total`, `limit`, and `offset`. |
| `update <id> [--title <text>] [--label k=v ...] [--unset-label <key> ...] [--reference <ref> ...] [--clear-references] [--trunk[=false]] [--auto-approve[=false]]` | Patch metadata in any state. References replace the list. See the invariant above before changing trunk mode. |
| `review approved <id> [--comment <text>]` | Record a human approval after the human explicitly supplies it. |
| `review rejected <id> --reason <text>` | Record a human rejection after the human explicitly supplies it. |
| `abandoned <id> --reason <text>` | Permanently abandon a task; a human-authorized decision. |
| `migrate [--dry-run]` | Convert legacy task files; run the dry run first. |
| `delete <id>` | Delete one task file. Human-authorized housekeeping, not part of the driver loop. |
| `prune [--dry-run]` | Delete all completed task files. Preview first; not part of the driver loop. |

Global CLI flags are `--git-dir` (default `.`), `--tasks-dir` (default `.tasks`),
`--worktrees-dir` (default `.worktrees`), `--json`, `--log-level`, and `--log-format`.

## Recover from errors

Exit code 1 is a failed precondition or domain error, such as a missing task, dirty repository, or
invalid transition. Exit code 2 is malformed input, such as a missing id or invalid check value.
Correct the named input and retry. If a transition is refused, call `next` again; do not force the
state or edit YAML.
