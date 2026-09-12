# taskman

**A deterministic workflow driver for coding agents.**

Taskman stores the durable state of a coding task and tells an agent what to do next. The agent
edits code, runs checks, reviews, commits, and merges. Taskman validates the workflow and records
progress - it does not perform any of that work itself.

The workflow is declared in Go and compiled into the binary. It is intentionally a concrete coding
workflow, not a generic YAML workflow engine.

> **Status:** alpha. Expect breaking changes and bugs.

## Install

Taskman requires Go 1.27+ and `git` on `PATH`.

```bash
go install github.com/mhmdkzr/taskman@latest
```

Tasks are stored in a single SQLite file, `./tasks.db` by default. Keep it out of version control:

```gitignore
tasks.db
```

## Use

Create a task:

```bash
taskman create \
  --title "Add rate limiting" \
  --description "Add per-client rate limiting to the HTTP API"
```

`create` prints the generated task id. Give that id to an agent:

```text
Use taskman to perform task <id>. Read the instruction each command returns and follow it until
it says to wait or the task is done.
```

There is no separate "what's next" command. Every command that reports an event - and `get` -
returns the task's current state and a derived instruction:

```json
{
  "state": "implement",
  "instruction": { "state": "implement", "action": "dispatch" }
}
```

`instruction.action` is one of:

- `dispatch`: agent work is required (write spec, implement, fix, review);
- `run`: perform a mechanical step such as checks or merge;
- `wait`: a human or external change is required;
- `done`: the task is completed or abandoned.

Run `taskman skill` to print Taskman's full agent operating instructions.

## Workflow

The main path is:

```text
specify → specification_review → implement → verify → automated_review → commit → human_review → merge → completed
```

Each task has one authoritative workflow state, derived by replaying its recorded events. Every
review gate, and the whole `verify` step, is optional per task:

- Verification checks are declared at `implemented` (`--unit`, `--integration`, `--end-to-end`,
  `--linters`). A task that declares none skips `verify`.
- Review gates are declared with `--agent-review` / `--human-review` at `specified` (specification
  gates) or `implemented` (implementation gates). Any task with a review gate must declare at
  least one verification check.
- A task always proceeds through `commit → merge` to `completed`; there is no shortcut that skips
  merge.

Failed verification or a rejected implementation review enters a fix state that loops back through
another verification attempt. A rejected specification returns to `specify` for revision.

### Specify

Without a specification, the task's instruction is `dispatch` at `specify`. Record the drafted
plan and choose its review gates:

```bash
taskman specified --id <id> --plan "..." [--agent-review] [--human-review]
```

`specification_review` is a single state covering both gates, and its instruction is always
`wait`. If an agent review is required, dispatch an independent reviewer and report it; the task
only truly waits on a human once any required agent gate is satisfied:

```bash
taskman specification review agent approved --id <id> [--comment "..."]
taskman specification review agent rejected --id <id> --finding <location>=<detail> ...

taskman specification review human approved --id <id> [--comment "..."]
taskman specification review human rejected --id <id> --reason "..."
```

### Implement and verify

Do the work in your own worktree and branch, then report them once:

```bash
taskman implemented --id <id> --worktree <path> --branch <name> \
  [--unit] [--integration] [--end-to-end] [--linters]
```

If checks are required, the task moves to `verify` (`run`). Run the checks and report the
outcome; a pass or fail is derived from whether any reported check is `error`:

```bash
taskman verified --id <id> --unit ok --linters error
```

Failed checks enter `fix_verification_failure` with no fixed retry limit.

### Automated review

If required, the implementation enters `automated_review` (`dispatch`). Use an independent
reviewer with clean context, then report the verdict:

```bash
taskman implementation review agent approved --id <id> [--comment "..."]
taskman implementation review agent rejected --id <id> --finding <location>=<detail> ...
```

A rejection loops through `fix_automated_review_findings` → verify → `automated_review` again,
with no built-in round limit.

### Commit and human review

After automated review approval (or directly, when no review gate is configured), the state is
`commit` (`dispatch`). Create a conventional commit in the task's worktree and report it; Taskman
reads the commit from Git itself:

```bash
taskman committed --id <id>
```

If a human review is required, the task then waits in `human_review`. A human, or an agent
explicitly directed by one, reports:

```bash
taskman implementation review human approved --id <id> [--comment "..."]
taskman implementation review human rejected --id <id> --reason "..."
```

A rejection enters `fix_human_review_findings` → verify → new commit → `human_review` again. This
does not repeat automated review.

### Merge

Perform the actual merge yourself, into a target branch in the `--git-dir` repository, then report
it. Taskman reads the resulting commit from `--target`:

```bash
taskman merged --id <id> --target main
```

### Block and abandon

A dispatched worker that gives up can block the task:

```bash
taskman escalated --id <id> --stage implementation --reason "Required API behavior is ambiguous"
```

There is no resume command; a blocked task waits or can be abandoned. `abandoned` ends a task
unsuccessfully from any non-terminal state:

```bash
taskman abandoned --id <id> --reason "Feature is no longer required"
```

`completed` and `abandoned` are terminal.

## Output

Every command that returns a task can render it three ways:

- default: a one-line human-readable summary;
- `--json`: the task plus its derived `state` and `instruction`;
- `--md`: a full Markdown document.

`--json` and `--md` are mutually exclusive. Global flags are:

```text
--git-dir <path>    repository root taskman reads commits from (default ".")
--db <path>         SQLite task database (default "./tasks.db")
--json              print the JSON envelope
--md                render a Markdown document
--log-level <lvl>   debug, info, warn, or error (default "info")
--log-format <fmt>  text or json (default "text")
```

## MCP

`taskman mcp` serves the same operations as MCP tools over stdio, using the root flags bound at
startup. Tools are named `task_<command>` - for example `task_get`, `task_committed`,
`task_specification_review_agent_approved` - and each returns the same JSON document as `--json`.

## Storage

Tasks are stored as an append-only event log in a single SQLite file (`--db`, default
`./tasks.db`). The current state is never stored directly; it is derived by replaying the task's
events. Treat the schema as private: use `get`, `list`, and the reporting commands rather than
opening the database yourself.

## Architecture

Taskman uses a functional core, imperative shell:

```text
CLI or MCP request
        ↓
command shell: validate input and obtain external facts
        ↓
typed event
        ↓
task.Apply(current, event) → updated task
        ↓
SQLite persistence (validated in a transaction)
```

- `internal/task` is the pure core: the `Task` aggregate, its append-only state history, the closed
  set of event types, the compiled workflow, `Apply`, and `Instruction`. It performs no
  filesystem, Git, clock, logging, CLI, MCP, or rendering work.
- `internal/task/store` is the SQLite-backed event store (`Open`, `Create`, `Read`, `Append`,
  `List`).
- `internal/git` is the imperative Git adapter used by command shells to read commits.
- `internal/utils` holds the CLI plumbing shared by every slice, including the `--json`/`--md`
  renderers and exit-code mapping.
- `internal/commands/<slice>` is one vertical slice per command, with its domain logic, CLI
  frontend (`cmd.go`), MCP frontend (`mcp.go`, where wired up), and README together. The two
  frontends are thin, independent callers of the same application function.
- `internal/cli` assembles the root command; `internal/mcp` assembles the MCP server.

Run `taskman --help` or `taskman <command> --help` for the complete command reference.

## Development

```bash
make fmt
make lint
make test
make build
```

## License

MIT
