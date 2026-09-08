---
name: taskman
description: Drive taskman task lifecycles - task create, task next, task verify, task review record/approve/reject, task commit, task merge, task escalate, .tasks/*.yaml files. Use when creating or working a taskman task, when a taskman response (dispatch/run/wait/done, report_with) tells you to report back, or when developing or debugging taskman itself.
---

# taskman

taskman is a file-backed, one-shot CLI task server: it holds tasks as `.tasks/*.yaml` files and
validates and records the state transitions you report to it. It executes almost nothing itself -
no builds, no commits, no merges. You (or a sub-agent you dispatch) do the work, then tell taskman
what happened. Full design: `notes/design/design.md`.

## Invocation

Use `taskman` if it is on PATH; otherwise, from this repo's checkout:

```bash
go run . task <command> ...
```

Global flags (apply to every command): `--git-dir` (default `.`), `--tasks-dir` (default
`.tasks`), `--worktrees-dir` (default `.worktrees`), `--json` (full JSON envelope instead of a
human-readable summary), `--log-level`, `--log-format`.

## The core loop

For any task id, `taskman task next <id>` tells you what to do. Branch on its `action` field:

- **`dispatch`** - judgment work is needed. Do the work described in `message`, working in the
  worktree and branch the message names. When done, report back with the command given in
  `report_with`.
- **`run`** - a mechanical step (build checks, or the merge). Run it yourself as `message`
  describes, then report with `report_with`.
- **`wait`** - a human gate (review pending, or task blocked). Stop; do nothing further. Never
  approve, reject, or merge on a human's behalf.
- **`done`** - the task is completed or failed. Drop it.

A caller that only ever calls `task next`, does what `message` says, and reports with
`report_with` is, by construction, driving the state machine correctly.

## Command reference

| Command | Purpose |
|---|---|
| `task create --definition <text> [--title <text>] [--id <id>] [--label k=v ...] [--reference <ref> ...] [--specification <text> --done-when <text>]` | Create a task. Requires a **clean working tree**. Creates the worktree `.worktrees/<id>` on branch `task/<id>` itself - the one thing taskman executes. `.worktrees/` must be gitignored first. With `--specification`/`--done-when`, the specify stage is skipped. |
| `task next <id>` | Ask what to do next (see core loop). |
| `task get <id>` / `task list [--state <state> ...] [--label k=v ...]` | Read tasks. Never parse the YAML by hand for decisions. |
| `task update <id> [--title ...] [--label k=v ...] [--unset-label k ...] [--reference <ref> ...] [--clear-references]` | Patch metadata only - safe on tasks in any state. |
| `task specify <id> --result <text> --done-when <text>` | Record a drafted specification and acceptance criteria. |
| `task implement <id>` | Record that an implementation attempt exists (a diff in the worktree). |
| `task verify <id> --check <name>=<ok\|error> ... [--output <text>]` | Report one build-check attempt. One `--check` per check actually run (e.g. `--check vet=ok --check test=ok`); the attempt passes only if every check is `ok`. |
| `task review record <id> --approved <bool> [--finding <file>=<text> ...]` | Report the **automated** review round's verdict. Findings carry the full detail text, not summaries. |
| `task commit <id> [--commit <hash>]` | Report a commit you already made (see committing below). |
| `task escalate <id> --stage <stage> --reason <text>` | Report that the dispatched agent gave up (called its escalate tool). Blocks the task. |
| `task review approve <id> [--comment <text>]` | **Human** approval only - never call this yourself. |
| `task review reject <id> --reason <text>` | **Human** rejection only - never call this yourself. |
| `task merge <id> [--commit <hash>]` | Report a merge you already made. |
| `task abandon <id> --reason <text>` | Mark the task failed for good. Human decision. |
| `task delete <id>` | Housekeeping. Never invoke as part of driving a task. |

## Lifecycle and rules

Stages run `definition → specification → implementation → verification → review → merge`.

- **Never edit `.tasks/*.yaml` by hand.** Every write goes through a taskman command.
- **Work in the worktree** `.worktrees/<id>` on branch `task/<id>` - created by `task create`,
  left in place for the task's whole life.
- **Verification loop** (two automated rounds max): run build checks → `task verify`. Pass →
  dispatch an automated reviewer with the task's `specification` and `done_when` as acceptance
  criteria → `task review record`. Rejected → dispatch a fix agent with the failure output or
  findings, then `task verify` again. A second automated rejection blocks the task automatically;
  `task next` will then say `wait`.
- **Escalation**: if the dispatched agent gives up instead of iterating, report it with
  `task escalate --stage <stage> --reason <text>`. There is no numeric retry cap on the
  build-fix loop - escalation is the only bound.
- **Committing**: when `task next` says to draft a commit (right after automated review
  approves), write a conventional-commit-style message (`feat:`, `fix:`, ...), run `git commit`
  **yourself** in the worktree (new commit, never an amend), then report it with
  `task commit <id>`. taskman reads the real hash and message back via `git log` - it does not
  trust reported text.
- **Human review**: after `task commit`, `task next` says `wait` until a human runs
  `task review approve` or `task review reject`.
- **Review-reject recovery**: on a human rejection, fix, re-verify, make **another new commit**,
  and `task commit` again. This cycle skips the automated reviewer - the human is now the
  reviewer. Each cycle adds one commit; never amend.
- **Merge**: `task next` says `run` - merge branch `task/<id>` into the base branch yourself,
  then `task merge <id>`.

## Errors and exit codes

Exit 1 means a failed precondition or domain error (`task not found`, an invalid transition -
the message states which precondition failed), or a dirty working tree on `task create`. Exit 2
means malformed input (bad label, bad `--check` value, missing id). Read the message, fix what it
names, and retry the same command; if a transition was genuinely refused, re-check with
`task next` instead of forcing it.
