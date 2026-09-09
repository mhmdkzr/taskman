---
name: taskman
description: Drive taskman task lifecycles - create, next, verify, review record/approve/reject, commit, merge, escalate, .tasks/*.yaml files. Use when creating or working a taskman task, when a taskman response (dispatch/run/wait/done, report_with) tells you to report back, or when developing or debugging taskman itself.
---

# taskman

taskman is a file-backed, one-shot CLI task server: it holds tasks as `.tasks/*.yaml` files and
validates and records the state transitions you report to it. It executes almost nothing itself -
no builds, no commits, no merges. You (or a sub-agent you dispatch) do the work, then tell taskman
what happened. Full design: `notes/design/design.md`.

## Invocation

`taskman` should be on your PATH.

Global flags (apply to every command): `--git-dir` (default `.`), `--tasks-dir` (default
`.tasks`), `--worktrees-dir` (default `.worktrees`), `--json` (full JSON envelope instead of a
human-readable summary), `--log-level`, `--log-format`.

## The core loop

For any task id, `taskman next <id>` tells you what to do. Branch on its `action` field:

- **`dispatch`** - judgment work is needed. Do the work described in `message`, working in the
  worktree and branch the message names. When done, report back with the command given in
  `report_with`.
- **`run`** - a mechanical step (build checks, or the merge). Run it yourself as `message`
  describes, then report with `report_with`.
- **`wait`** - a human gate (review pending, or task blocked). Stop; do nothing further. Never
  approve, reject, or merge on a human's behalf.
- **`done`** - the task is completed or failed. Drop it.

A caller that only ever calls `next`, does what `message` says, and reports with
`report_with` is, by construction, driving the state machine correctly.

## Command reference

| Command | Purpose |
|---|---|
| `create --definition <text> [--title <text>] [--id <id>] [--label k=v ...] [--reference <ref> ...] [--specification <text> --done-when <text>] [--trunk] [--auto-approve]` | Create a task. Requires a **clean working tree**. Creates the worktree `.worktrees/<id>` on branch `task/<slug>` itself - the one thing taskman executes. `.worktrees/` and `.tasks/*.lock` must both be gitignored first, or a leftover worktree or lock file from a previous task fails this precondition on the next one. With `--trunk`, skips the worktree/branch and records the repo root and current branch instead - the task is worked in place. With `--specification`/`--done-when`, the specify stage is skipped. With `--auto-approve`, `commit` completes the review stage on its own - see below. |
| `next <id>` | Ask what to do next (see core loop). |
| `get <id>` | Read one task. Prefer relying on `next`'s own message instead - it already carries the definition, specification, done_when and references you need for the current step, so `get` is usually not necessary mid-flow. |
| `list [--state <state> ...] [--label k=v ...] [--limit <n>] [--offset <n>]` | List/filter tasks, paginated (`--limit` defaults to 50 to avoid dumping a huge tasks-dir; `0` means unlimited). `--json` returns `{tasks, total, limit, offset}` - use `total` to know whether more pages remain, and `--offset` to page through them. Never parse the YAML by hand for decisions. |
| `update <id> [--title ...] [--label k=v ...] [--unset-label k ...] [--reference <ref> ...] [--clear-references] [--trunk[=false]] [--auto-approve[=false]]` | Patch metadata only - safe on tasks in any state. `--trunk`/`--auto-approve` set the same flags `create` does; pass `=false` to unset either one. |
| `specify <id> --result <text> --done-when <text>` | Record a drafted specification and acceptance criteria. |
| `implement <id>` | Record that an implementation attempt exists (a diff in the worktree). |
| `verify <id> --check <name>=<ok\|error> ... [--output <text>]` | Report one build-check attempt. One `--check` per check actually run (e.g. `--check vet=ok --check test=ok`); the attempt passes only if every check is `ok`. |
| `review record <id> --approved <bool> [--finding <file>=<text> ...]` | Report the **automated** review round's verdict. Findings carry the full detail text, not summaries. |
| `commit <id> [--commit <hash>]` | Report a commit you already made (see committing below). |
| `escalate <id> --stage <stage> --reason <text>` | Report that the dispatched agent gave up (called its escalate tool). Blocks the task. |
| `review approve <id> [--comment <text>]` | **Human** approval only - never call this yourself. For a task created with `--auto-approve`, `commit` already completed review; this command has nothing left to do there. |
| `review reject <id> --reason <text>` | **Human** rejection only - never call this yourself. |
| `merge <id> [--commit <hash>]` | Report a merge you already made. |
| `abandon <id> --reason <text>` | Mark the task failed for good. Human decision. |
| `delete <id>` | Housekeeping. Never invoke as part of driving a task. |

## Lifecycle and rules

Stages run `definition → specification → implementation → verification → review → merge`.

- **Never read or edit `.tasks/*.yaml` by hand** - not with `cat`/`read`/an editor, not even to
  "just peek". Every read goes through `get`/`list`/`next`; every write goes through a taskman
  command. The file format is an implementation detail and can change.
- **Work in the worktree named by the task's `git.worktree`/`git.branch` fields** (`get <id>`
  or any command's output names them) - `.worktrees/<id>` on branch `task/<slug>` by default, created
  by `create` and left in place for the task's whole life; the repo root on the current branch
  if the task was created with `--trunk`. Don't assume the default path - read it from the task.
- **Verification loop** (two automated rounds max): run build checks → `verify`. Pass →
  dispatch an automated reviewer with the task's `specification` and `done_when` as acceptance
  criteria → `review record`. **The reviewer must be a separate subagent, not you and not the
  agent that implemented or fixed the code** - spawn it (e.g. via your Task/Agent tool) with a
  clean context containing only the `specification`, `done_when`, and the diff/commit to review,
  never the implementer's conversation history or reasoning. Reviewing your own work in the same
  context defeats the point of the review stage. Rejected → dispatch a fix agent with the failure
  output or findings, then `verify` again. A second automated rejection blocks the task
  automatically; `next` will then say `wait`.
- **Escalation**: if the dispatched agent gives up instead of iterating, report it with
  `escalate --stage <stage> --reason <text>`. There is no numeric retry cap on the
  build-fix loop - escalation is the only bound.
- **Committing**: when `next` says to draft a commit (right after automated review
  approves), write a conventional-commit-style message (`feat:`, `fix:`, ...), run `git commit`
  **yourself** in the worktree (new commit, never an amend), then report it with
  `commit <id>`. taskman reads the real hash and message back via `git log` - it does not
  trust reported text. **Stage explicitly** (`git add <files you actually touched>`), never `git
  add -A`/`git commit -a` - every taskman command rewrites the task's own YAML in place, and in
  `--trunk` mode that file lives in the same working tree as your change (plus, in trunk mode
  specifically, whatever else happens to be sitting uncommitted in that shared tree), so a blanket
  add can sweep in state you didn't mean to commit.
- **Verifying a doc-only or no-op change**: `verify` still expects at least one `--check`.
  There's no dedicated "no build" check name - use one that reflects what you actually confirmed
  (e.g. `--check review=ok` for a read-through, or a real command like `--check
  grep-stale-refs=ok` for a targeted search), not a placeholder that claims a build ran when none
  did.
- **Human review**: after `commit`, `next` says `wait` until a human runs
  `review approve` or `review reject`. Exception: a task created with `--auto-approve` never
  reaches this wait at all - `commit` completes review in the same call that records the commit,
  since the automated round already run inside verification is the only review that task gets.
- **Review-reject recovery**: on a human rejection, fix, re-verify, make **another new commit**,
  and `commit` again. This cycle skips the automated reviewer - the human is now the
  reviewer. Each cycle adds one commit; never amend.
- **Merge**: `next` says `run` - merge the task's branch into the base branch yourself, then
  `merge <id>`. A task created with `--trunk` never reaches this step - there's nothing to merge
  (the branch is already the target), so it completes automatically the moment review does.
- **Final bookkeeping commit**: the task's own `.tasks/<id>.yaml` keeps changing after the code
  commit (through review and merge), so it can never be part of that commit. taskman commits it
  itself, the moment the task goes terminal (`merge`, `abandon`, or an auto-approve/trunk
  completion inside `commit`/`review approve`) - there's nothing for you to do here.

## Errors and exit codes

Exit 1 means a failed precondition or domain error (`task not found`, an invalid transition -
the message states which precondition failed), or a dirty working tree on `create`. Exit 2
means malformed input (bad label, bad `--check` value, missing id). Read the message, fix what it
names, and retry the same command; if a transition was genuinely refused, re-check with
`next` instead of forcing it.
