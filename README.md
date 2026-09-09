# taskman

Taskman let's you define tasks and guides AI agents through their lifecycle until their completion. After a task is defined, the agent would simply run `taskman next <id>` and taskman will tell the agent exactly what to do next, and keeps track of the state transitions in the task files as the task progresses. Taskman has CLI and MCP interfaces. Taskman is currently in alpha phase, expect breaking changes and potential bugs.

## A task

A task is one unit of work, serialized as a YAML file in the tasks directory (default
`.tasks/<id>.yaml`, where `id` is `<uuid-v7>_<slug>`). It tracks:

- a coarse state: `created`, `started`, `blocked`, `completed`, or `failed`;
- the `definition`, the drafted `specification`, and the `done_when` acceptance criteria;
- free-form `labels` and `references`;
- per-stage progress across the lifecycle stages `definition`, `specification`,
  `implementation`, `verification`, `review`, and `merge`;
- the git `worktree`/`branch` the task is worked on, and its recorded `commit`;
- append-only logs of verification attempts, automated review rounds, and human reviews;
- block/failure details, when applicable.

Concurrent writers are handled with per-task file locks, so multiple processes can drive
different tasks in the same repo safely.

## Git integration

Creating a task also creates an isolated git worktree and branch under the worktrees
directory (default `.worktrees`), guarded by a clean-working-tree check - use `--trunk` to
work the task on the current branch instead. taskman reads a commit back from the worktree
rather than trust a caller's report of it, and, once a task goes terminal (`merge`, `abandon`,
or an auto-approve/trunk completion), commits the task's own now-final file itself - the one
commit nothing else can attribute to, since that file keeps changing through review and merge.
Everything else (edits, checks, the code commit itself, the merge) is done by the operator
agent and merely reported to taskman. `git` must be on `PATH`.

Both `.worktrees/` and `.tasks/*.lock` (taskman's per-task file lock) must be gitignored, or a
leftover one from a previous task fails the next `create`'s clean-working-tree check.

## Installing

Requires Go 1.27+. From a checkout of this repo:

```
go install .
```

or to grab a released version directly:

```
go install github.com/mhmdkzr/taskman@latest
```

## Driving a task

Use `taskman list` to get list of tasks, once a task is chosen, the primary interface is `taskman next <id>`. It inspects a task's current state and
answers with an action plus, where relevant, the exact command to run next:

| Action | Meaning |
| --- | --- |
| `dispatch` | Send an agent to do a piece of work; report the outcome back with the listed command |
| `run` | Run the mechanical step yourself (a build check, the merge) and report the result |
| `wait` | Nothing to do right now - the task is blocked or waiting on a human review |
| `done` | The task reached `completed` or `failed`; nothing more to do |

`next` is the driver loop: call it, follow its instruction, report back, call it again,
until it says `done`.

By default the guidance is printed as natural language; `--json` returns a machine-readable
envelope (the `action` and `report_with` fields are what a non-LLM driver loop keys on).
To use with AI agents, you can run `taskman skill` to print taskman's own agent-facing driver skill, which teaches an agent how to drive tasks with these commands.

## The lifecycle

A task moves through six stages in order - definition, specification, implementation,
verification, review, merge. Each stage is completed by the command that records its
outcome, and taskman refuses any transition that doesn't follow from the current state.

1. **create** - record what the task should accomplish and create its worktree/branch.
   Pass `--label key=value`, `--reference path`, a fixed `--id`, or a `--title`. Provide
   `--specification` plus `--done-when` up front to skip the specification stage entirely,
   or `--auto-approve` to skip the human review gate. The task is created with the
   definition stage done.
2. **specify** - draft the task's `specification` and `done_when` acceptance criteria from
   its definition and record them (`specify <id> --result <text> --done-when <text>`).
   This starts the task.
3. **implement** - mark the implementation attempt as done after doing the work in the
   task's worktree (`implement <id>`).
4. **verification** - run build checks and report the outcome
   (`verify <id> --check name=ok|error ... [--output <text>]`). Each attempt is
   appended to the task's verification log. When the checks pass, an automated review
   round runs and its verdict is recorded (`review record <id> --approved <bool>
   [--finding file=<detail>]`). A failed check or a rejected review round loops back to
   fix and re-verify.
5. **review** - once the automated round approves, commit the work in the worktree and
   record it (`commit <id> [--commit <hash>]`). If the task was created with
   `--auto-approve`, review completes right there - the automated round it already went
   through is the only review this task gets, so there's no separate approval step.
   Otherwise a human reviews the branch and records their decision with `review approve
   <id> [--comment <text>]` or `review reject <id> --reason <text>`. A rejection starts a
   recovery loop: fix, re-verify, fresh commit, re-review.
6. **merge** - merge the task's branch back into the target and record it
   (`merge <id> [--commit <hash>]`). The task is now `completed`. A task created with
   `--trunk` has no real merge to do - it completes automatically the moment review does.

Two commands leave the normal flow: `escalate <id> --stage <stage> --reason <text>`
blocks a task because a dispatched agent gave up (state `blocked`), and
`abandon <id> --reason <text>` marks a task failed for good (state `failed`). A task
file can also be removed outright with `delete <id>`, a human housekeeping action.

## Command reference

| Command | What it does |
| --- | --- |
| `list` | list tasks, optionally filtered by state/label and paginated |
| `get <id>` | show one task |
| `next <id>` | show what should happen next for this task |
| `create` | create a task and its worktree/branch (`--trunk` to work it in place) |
| `update <id>` | patch a task's title, labels, references, trunk, or auto-approve setting |
| `specify <id>` | record the drafted specification and acceptance criteria |
| `implement <id>` | mark the implementation attempt as done |
| `verify <id>` | report one build-check attempt |
| `review record <id>` | report the automated review round's verdict |
| `review approve <id>` | record a human's approval |
| `review reject <id>` | record a human's rejection and start recovery |
| `commit <id>` | read back the commit that was already made |
| `escalate <id>` | block the task because a dispatched agent gave up |
| `merge <id>` | record that the task's branch was already merged |
| `abandon <id>` | mark the task failed for good |
| `delete <id>` | remove the task file outright |
| `skill` | print the agent-facing driver skill to stdout |

Each task file is validated and mutated under lock, and invalid transitions are rejected
with a distinct exit code rather than silently ignored. Every command's flags have `--help`.

## Global options

These apply to every command:

| Flag | Default | Purpose |
| --- | --- | --- |
| `--git-dir` | `.` | repository root taskman operates against |
| `--tasks-dir` | `.tasks` | directory holding task files |
| `--worktrees-dir` | `.worktrees` | directory `create` creates worktrees under |
| `--json` | off | print the full JSON envelope instead of a human-readable summary |
| `--log-level` | `info` | `debug`, `info`, `warn`, or `error` |
| `--log-format` | `text` | `text` or `json` |

## Repository layout

- `main.go` - thin entrypoint that calls `internal/cli`.
- `internal/task` - the `Task` domain type and its file-backed persistence primitives
  (`ReadTask`, `WriteTaskFile`, `MutateTask`), and the `GitClient`.
- `internal/commands` - one package per command slice (`create`, `get`, `list`, ...; the review
  stage nested further under `internal/commands/review`), each owning its CLI frontend (`cmd.go`),
  domain logic (`<name>.go`), dispatch prompts, and, where wired up, its MCP frontend (`mcp.go`,
  exporting `RegisterMCP()`).
- `internal/cli` - the root command: `cli.go` mounts every slice directly on the root `taskman`
  command - there's no intermediate grouping command.
- `internal/mcp` - the MCP frontend's assembler: mounts each slice's own `RegisterMCP` into an
  MCP/stdio server, served by the `taskman mcp` command (a regular slice, `internal/commands/mcp`).
- `internal/utils` - plumbing shared by every slice's `cmd.go` (flag parsing, output
  rendering, error-to-exit-code mapping) without importing back into `internal/cli`.
- `scripts` - markdown formatting used by `make fmt`.

See the package READMEs under `internal/` for the details of each layer.

## Development

```
make build   # compile check
make test    # full test suite
make lint    # vet, staticcheck, golangci-lint, govulncheck
make fmt     # gofmt + goimports + markdown formatting
```
