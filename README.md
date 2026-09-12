# taskman

Taskman provides a deterministic workflow driver for coding agents to perform generic pre-defined programming tasks. After a task is defined, the agent can simply call `taskman next <id>` and taskman will simply tell the agent what to do next based on the current state of the task and how it is configured.

> **Status:** alpha. Expect breaking changes and potential bugs.

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

Add taskman skill to your agent's skills. You can use `taskman skill` command to print it to stdout.

Create a task:

```bash
taskman create \
  --title "Add rate limiting" \
  --description "Add per-client rate limiting to the HTTP API"
```

`create` prints the generated task id. Give that id to an agent:

```text
Use taskman to perform task <id>. Follow taskman instructions.
```

`taskman next --id <id>` renders that instruction as guidance: a message built from the task's own
data (its plan, its worktree/branch, a failed check's output, a rejected review's findings) and
the command(s) that would currently report an outcome. 

## Workflow

The main path is:

```text
specify → specification_review → implement → verify → automated_review → commit → human_review → merge → completed
```

Every command that returns a task can render it three ways:

- default: a one-line text summary;
- `--json`: the task plus its derived `state` and `instruction`;
- `--md`: a full Markdown document.

`--json` and `--md` are mutually exclusive. See [Commands](#commands) for the global flags every
command accepts, and the full per-command flag reference.

## Commands

Every command accepts these global flags, in addition to any command-specific ones listed below:

```text
--git-dir <path>    repository root taskman reads commits from (default ".")
--db <path>         SQLite task database (default "./tasks.db")
--json              print the JSON envelope
--md                render a Markdown document
--log-level <lvl>   debug, info, warn, or error (default "info")
--log-format <fmt>  text or json (default "text")
```

```text
taskman create
  --title string                     short human-readable title
  --description string               what the task should accomplish
  --label string [--label string]    a label as key=value - repeatable

taskman specified
  --id string                             the task whose specification was submitted
  --plan string                           the specification's plan
  --agent-review                          require an automated review
  --agent-review-use-subagent              run the automated review in a subagent
  --agent-review-auto-fix                  automatically fix automated review findings
  --agent-review-auto-fix-max-rounds int   max automated-review auto-fix rounds (default 0)
  --agent-review-auto-fix-use-subagent     run automated-review auto-fix in a subagent
  --human-review                          require a human review
  --human-review-auto-fix                  automatically fix human review findings
  --human-review-auto-fix-max-rounds int   max human-review auto-fix rounds (default 0)
  --human-review-auto-fix-use-subagent     run human-review auto-fix in a subagent

taskman specification review agent approved
  --id string       the task whose specification's automated review was approved
  --comment string  an optional approval comment

taskman specification review agent rejected
  --id string                            the task whose specification's automated review was rejected
  --finding string [--finding string]    a finding as location=detail - repeatable

taskman specification review human approved
  --id string       the task whose specification's human review was approved
  --comment string  an optional approval comment

taskman specification review human rejected
  --id string      the task whose specification's human review was rejected
  --reason string  why the specification's human review was rejected

taskman implemented
  --id string                             the task that was implemented
  --worktree string                       the worktree the implementation was done in
  --branch string                         the branch the implementation was done on
  --unit                                  require unit tests
  --integration                          require integration tests
  --end-to-end                           require end-to-end tests
  --linters                               require linters
  --verification-auto-fix                 automatically fix verification failures
  --verification-auto-fix-max-rounds int   max verification auto-fix rounds (default 0)
  --verification-auto-fix-use-subagent     run verification auto-fix in a subagent
  --agent-review                          require an automated review
  --agent-review-use-subagent              run the automated review in a subagent
  --agent-review-auto-fix                  automatically fix automated review findings
  --agent-review-auto-fix-max-rounds int   max automated-review auto-fix rounds (default 0)
  --agent-review-auto-fix-use-subagent     run automated-review auto-fix in a subagent
  --human-review                          require a human review
  --human-review-auto-fix                  automatically fix human review findings
  --human-review-auto-fix-max-rounds int   max human-review auto-fix rounds (default 0)
  --human-review-auto-fix-use-subagent     run human-review auto-fix in a subagent

taskman implementation review agent approved
  --id string       the task whose implementation's automated review was approved
  --comment string  an optional approval comment

taskman implementation review agent rejected
  --id string                            the task whose implementation's automated review was rejected
  --finding string [--finding string]    a finding as location=detail - repeatable

taskman implementation review human approved
  --id string       the task whose implementation's human review was approved
  --comment string  an optional approval comment

taskman implementation review human rejected
  --id string      the task whose implementation's human review was rejected
  --reason string  why the implementation's human review was rejected

taskman verified
  --id string           the task whose verification was reported
  --unit string         the unit test check's result: ok or error
  --integration string  the integration test check's result: ok or error
  --end-to-end string   the end-to-end test check's result: ok or error
  --linters string      the linters check's result: ok or error
  --output string       verification output/log text

taskman committed
  --id string  the task whose commit was recorded

taskman merged
  --id string      the task that was merged
  --target string  the branch it was merged into

taskman escalated
  --id string      the task being escalated
  --stage string   the stage the task is stuck at
  --reason string  why the task is stuck

taskman abandoned
  --id string      the task being abandoned
  --reason string  why the task is being abandoned

taskman get
  --id string  the task id to read

taskman list
  (no command-specific flags)

taskman next
  --id string  the task id to inspect

taskman skill
  (no command-specific flags; prints SKILL.md to stdout)

taskman mcp
  (no command-specific flags; serves task operations over MCP/stdio)
```

## MCP

`taskman mcp` serves the same CLI operations as MCP tools over stdio, using the root flags bound at
startup. Tools are named `task_<command>` - for example `task_get`, `task_committed`,
`task_specification_review_agent_approved` - and each returns the same JSON document as `--json`.

## Storage

Tasks are stored as an append-only event log in a single SQLite file. 
The current state is never stored directly; it is derived by replaying the task's events. 

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
