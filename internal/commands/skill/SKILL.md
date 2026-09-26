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
short summary plus the rendered guidance; the envelope carries the task plus its current `state`,
derived `instruction: {state, action}`, a `message` built from the task's own data (its plan,
worktree/branch, a failed check's output, a rejected review's findings), and the exact
`commands` that would currently report an outcome - every one of them, if more than one gate is
pending at once (see the `specification_review` nuance below). Read `instruction.action` to know
what to do next:

- `dispatch`: judgment work is required at this state. Perform or delegate it, then report the
  outcome with the matching command below.
- `run`: perform the described mechanical operation (running verification, merging) then report
  the outcome.
- `wait`: stop. This state needs a decision or event from outside the agent's control.
- `done`: stop. The task is completed or abandoned.

The guidance performs no transition and never decides a review's verdict or a check's result for
you - read what's expected from `message`, then act and report through the matching `commands`
entry.

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

## Creating a task: the intake interview

`create` and `specified` between them capture every substantive fact taskman ever records about
a task: title, description, labels, the specification plan, and every review/verification gate.
(Labels are the one field editable later too, via `label add`/`label remove` - see "Management
commands" - but still get them right at intake rather than leaning on that as a fallback.)
Taskman performs no judgment about any of these - they must come from an explicit human decision,
never from the agent guessing, inferring from a vague ask, or picking a convenient default and
moving on. Treat filling them in as a short conversation with the human, not a form to fill out
of your own initiative: respond to what they actually said, offer to draft things for them
(a title, a plan grounded in the real code) and let them approve or edit, but never finalize a
substantive field they haven't explicitly approved or written themselves.

Mix plain chat with a structured-choice prompt (e.g. `AskUserQuestion`, where the environment
provides one), chosen per question: use a structured choice when there's a real, bounded decision
(yes/no gates, which of a known set of labels applies, which verification checks apply); ask in
plain chat when it's open text with nothing to choose between (the description body, the plan
narrative). Don't force open text through a choice UI just for consistency, and don't invent
placeholder options for a question that has none. Batch independent structured questions together
rather than one at a time.

1. **Description before title.** Ask what the task is about before asking for a title - a name is
   easier to pick once the substance exists. From the description, draft two or three candidate
   titles yourself (short, imperative) and offer them for approval alongside an open option; don't
   settle on a title the user hasn't picked or written.
2. **Labels.** Don't invent a label taxonomy. If the project has one (check its own docs/CLAUDE.md
   for conventions - e.g. a module or component list, task-type or priority conventions), offer
   those as real options; otherwise ask the user directly what labels, if any, they want. Skip a
   key entirely rather than force a value that doesn't apply.
3. **The specification plan.** Ask for it, or offer to draft one by reading the relevant code
   yourself and proposing something concrete (real paths, real existing patterns) for approval -
   either way, the user must actually see and approve the final text, not just the intent behind
   it.
4. **Every review/auto-fix gate, in full.** As covered above, each gate is three independent
   settings - don't stop at asking for a round cap. For each applicable gate ask, in order:
   (a) is it required, and (for an agent-run gate) should it run in a subagent; (b) should
   rejections/failures be auto-fixed automatically at all, as its own explicit yes/no; (c) only if
   (b) is yes, the round cap and whether the fixing itself runs in a subagent. This applies to
   *all four* review gates `specified` now declares in one call: the specification's own
   agent-review/human-review (`--agent-review`/`--human-review`), and the eventual
   implementation's agent-review/human-review (`--impl-agent-review`/`--impl-human-review`).
5. **Verification checks and worktree/branch.** Suggest checks (`unit`/`integration`/`end-to-end`/
   `linters`) based on what the project's own testing conventions say about the kind of code being
   touched, and let the user confirm or edit - don't lock in a suggestion unconfirmed. A gate with
   any review requires at least one check; resolve that conflict with the user before moving on if
   it comes up. Also ask now whether implementation should happen in a fresh worktree/branch or the
   current checkout: `--use-worktree` (plus `--worktree <path>`/`--branch <name>`, required
   together with it) is set on `specified`, not `implemented` - deciding it here means `implemented`
   later needs no location re-entered; it copies it forward automatically (auto-detecting the
   current repository's worktree/branch instead, if `--use-worktree` was never set).
6. **Draft, then approve, then create.** Compile everything into one concrete draft - title,
   description, labels, the full plan text, and the exact state of every gate (required?
   subagent? auto-fix enabled? cap? auto-fix subagent?), every verification check, and the
   worktree decision - and show it before touching taskman. Loop on feedback until approved; call
   no taskman command during that loop.
7. **Create, specify, report, stop.** Once approved: `create` (recording the returned id), then
   `specified` with every flag the user actually chose - not just `--agent-review`/
   `--human-review` with a bare round-cap number, since a cap without its matching `-auto-fix`
   flag does nothing (see above). Report the new task's id, title, and current
   state/instruction back to the user, then **stop** - do not dispatch the specification review,
   do not proceed to `implemented`, and do not start implementation work. Handing back the id is
   the end of this interview; anything past it is a separate action the human asks for
   explicitly, through the ordinary driver loop above.

## Preserve these invariants

- Never open or edit the SQLite database file directly, even for inspection. Use `get`, `list`,
  and the reporting commands. The schema is private to Taskman.
- Taskman does not create or manage Git worktrees or branches. The worktree path and branch name
  are decided once, up front, at `specified --use-worktree --worktree <path> --branch <name>`
  (or the decision to use none at all); create the worktree and branch yourself at that exact
  location (however your environment normally does that), do the work there, then just call
  `implemented --id <id>` - no location to pass, it was already declared. `committed` then reads
  the current commit from that recorded worktree - do not pass a worktree path again. `merged` is
  different: it reads the resulting commit from the `--target` branch in the `--git-dir`
  repository, not from the task's worktree.
- Report facts only after they are true. `implemented` means an implementation attempt matching
  the declared policy now exists (in the declared worktree, if one was declared); `verified`
  describes checks actually run; `committed` reads a commit that already exists there; `merged`
  records a merge that has already happened.
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

- Verification failures and implementation-review rejections loop back through a fix state. Each
  of the three auto-fix-capable gates (verification, agent-review, human-review) is **three
  independent settings**, not one: `--{verification,agent-review,human-review}-auto-fix` is what
  actually turns on unattended automatic fixing; `--*-auto-fix-max-rounds` is the round cap, which
  is silently inert unless the matching `-auto-fix` flag is also set (a task with a cap but
  `auto-fix.enabled: false` in its JSON will loop unbounded, not stop at the cap - always pass
  both together); and `--*-auto-fix-use-subagent` decides whether the fixing runs in a subagent
  or the current session. A gate with auto-fix enabled and a cap blocks the task once the
  recorded rounds exceed it, instead of dispatching another fix. Without a cap (or without
  auto-fix enabled at all) the loop is unbounded - keep fixing and re-reporting, or use
  `escalated` if it is not converging.
- After automated review approval, create a new conventional commit in the task's worktree and
  report it with `committed`. On human review rejection, fix, verify again, create another new
  commit, and report it; do not amend. Human-review rejection recovery does not repeat automated
  review.
- Every review gate - specification agent/human, implementation agent/human - is independently
  optional per task, all four chosen at `specified` time (`--agent-review`/`--human-review` for
  the specification, `--impl-agent-review`/`--impl-human-review` for the eventual implementation),
  and a gate that is not required is refused by its `approved`/`rejected` commands. Every gate is
  (re)configured each time `specified` is called, so a task sent back to `specify` by a rejection
  can change any of them; a resubmission replaces the specification wholesale and clears prior
  review results. Any task with an implementation review gate must also declare at least one
  verification check.
- A task always proceeds through `commit` -> (human review, if required) -> `merge` to
  `completed` - there is no shortcut that skips merge.
- `escalated` blocks a non-blocked task and records where and why work stopped. It is for a
  dispatched worker giving up, not for an ordinary failed check that can be retried.
- `abandoned` ends a task unsuccessfully from any non-terminal state, with a free-text reason
  (there is no separate "kind" of failure to pick from - infeasible, no-longer-needed, superseded,
  etc. are all just what you write in `--reason`).
- `unblocked` resumes a blocked task at whatever state it occupied right before it blocked. For a
  budget-exhaustion blockage, `--rounds` must grant more auto-fix rounds to the exhausted gate
  (required, `> 0` - resuming without one just re-blocks on the next failure); for an
  `escalated` blockage `--rounds` must be omitted, since there is no budget to grant.

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

Once `merged` is recorded, the task is done and its worktree and branch are no longer needed -
clean them up yourself (Taskman does not): `git worktree remove <worktree-path>`, then `git branch -d <branch>`.

## Reporting commands

| Command | Meaning |
|---|---|
| `specified --id <id> --plan <text> [--agent-review ...] [--human-review ...] [--unit] [--integration] [--end-to-end] [--linters] [--verification-auto-fix] [--verification-auto-fix-max-rounds <n>] [--verification-auto-fix-use-subagent] [--impl-agent-review ...] [--impl-human-review ...] [--use-worktree --worktree <path> --branch <name>]` | Record the drafted specification, which of its own review gates are required, and every requirement for the eventual implementation: verification checks, implementation review gates, and worktree/branch policy. (`...` stands for each gate's own `-use-subagent`/`-auto-fix`/`-auto-fix-max-rounds`/`-auto-fix-use-subagent` flags.) |
| `specification review agent approved --id <id> [--comment <text>]` | Record an independent agent reviewer's approval of the specification. |
| `specification review agent rejected --id <id> --finding <location>=<detail> ...` | Record the agent reviewer's findings against the specification. |
| `specification review human approved --id <id> [--comment <text>]` | Record human approval of the specification. |
| `specification review human rejected --id <id> --reason <text>` | Record human rejection of the specification, for revision. |
| `implemented --id <id>` | Record that an implementation attempt is ready. Verification checks, review gates, and worktree/branch are copied forward from the specification automatically - nothing else to pass. |
| `verified --id <id> [--unit ok\|error] [--integration ok\|error] [--end-to-end ok\|error] [--linters ok\|error] [--output <text>]` | Record one verification attempt. Report every required check; whether it counts as a pass or a fail is derived from whether any reported check is `error` - you never say "passed" or "failed" directly. |
| `implementation review agent approved --id <id> [--comment <text>]` | Record an independent agent reviewer's approval of the implementation. |
| `implementation review agent rejected --id <id> --finding <location>=<detail> ...` | Record the agent reviewer's findings against the implementation. |
| `committed --id <id>` | Read and record the task worktree's current commit. |
| `implementation review human approved --id <id> [--comment <text>]` | Record human approval of the implementation. |
| `implementation review human rejected --id <id> --reason <text>` | Record human rejection of the implementation, for another fix-and-verify round. |
| `merged --id <id> --target <branch>` | Read and record a merge already performed into `target`. |
| `escalated --id <id> --stage <text> --reason <text>` | Block the task after dispatched work gives up. |
| `abandoned --id <id> --reason <text>` | End the task unsuccessfully; a human-authorized decision. |
| `unblocked --id <id> --reason <text> [--rounds <n>]` | Resume a blocked task, granting more auto-fix rounds for a budget-exhaustion blockage. |

Verification checks are declared at `specified`; a task that declares none skips `verify`
entirely (but any review gate forces at least one check). When `verify` is reached, report every
required check - `verified` refuses an attempt that reports none. Report what was actually
confirmed; do not claim a build or test that did not run.

## Management commands

| Command | Purpose |
|---|---|
| `create --title <text> --description <text> [--label k=v ...]` | Create a task. Returns its generated id. |
| `get --id <id>` | Read one task's current state, instruction, guidance, and the commands that can report its next outcome. |
| `list` | List tasks. Optional filters `--state <state>` and `--label <k=v>` (repeatable), and pagination `--limit <n>` (default 50; 0 = unlimited) / `--offset <n>`. |
| `label add --id <id> --label k=v [...]` | Set or overwrite one or more labels. |
| `label remove --id <id> --key k [...]` | Delete one or more labels; a key that isn't present is not an error. |
| `delete --id <id>` | Permanently delete a task and its event log. Irreversible. |
| `prune [--dry-run]` | Permanently delete every completed task and its event log. Irreversible. |

`label add`/`label remove` are the one pair of commands here that go through the same event log
and `Apply` as every reporting command above, but as a global, state-preserving transition: they
work from any non-terminal state and never change `state`/`instruction`, since labels are plain
metadata rather than part of the workflow. They still can't touch a `completed` or `abandoned`
task, same as everything else.

`delete` and `prune` are the management commands that destroy data. `delete --id` removes one
task and its entire event log from any state; `prune` removes every task in the `completed` state
(use `--dry-run` to see which ids would be removed first). After either, `get` reports the
task as not found and `list` no longer shows it. Taskman does not grant the authority to delete -
only run `delete` or `prune` when a human has explicitly authorized it.

Global CLI flags are `--git-dir` (default `.`), `--db` (default `./tasks.db`), `--json`,
`--md`, `--log-level`, and `--log-format`. `--json` and `--md` are mutually exclusive: they
select the machine-readable envelope or a full Markdown document, respectively, in place of the
default human-readable summary.

## Recover from errors

Exit code 1 is a failed precondition or domain error, such as a missing task or an invalid
transition. Exit code 2 is malformed input, such as a missing `--id` or an invalid check value.
Correct the named input and retry. If a transition is refused, re-read the task (`get --id <id>`)
and reconsider its `instruction` rather than forcing the state.
