# taskman

**A deterministic workflow driver for coding agents.**

Taskman stores the durable state of a coding task and tells an agent what to do next. The agent
edits code, runs checks, reviews, commits, and merges. Taskman validates the workflow and records
progress.

The workflow is declared in Go and compiled into the binary. It is intentionally a concrete
coding workflow, not a generic YAML workflow engine.

> **Status:** alpha. Expect breaking changes and bugs.

## Install

Taskman requires Go 1.27+ and `git` on `PATH`.

```bash
go install github.com/mhmdkzr/taskman@latest
```

Ignore worktrees and locks, but keep task YAML files tracked:

```gitignore
.worktrees/
.tasks/*.lock
```

## Use

Create a task:

```bash
taskman create \
  --title "Add rate limiting" \
  --definition "Add per-client rate limiting to the HTTP API"
```

Then give its ID to an agent:

```text
Use taskman to perform task <id>. Follow taskman next until it says to wait or the task is done.
```

The agent repeats:

```text
taskman next <id>
        ↓
follow the instruction
        ↓
report with the command provided by taskman
        ↓
repeat
```

`next` returns one of four actions:

- `dispatch`: agent work is required;
- `run`: perform a mechanical step such as checks or merge;
- `wait`: a human or external change is required;
- `done`: the task is completed or abandoned.

Run `taskman skill` to print Taskman's agent operating instructions.

## Workflow

The main path is:

```text
specify → specification_review → implement → verify → automated_review → commit → human_review → merge → completed
```

Each task has one authoritative workflow state.

State names describe work in progress; the commands that report a completed workflow fact use
past tense. For example, a task in `implement` becomes `verify` only after
`taskman implemented <id>` records that the implementation is complete.

### Specify, implement, and verify

If a task has no specification, `next` asks an agent to write one. Record it with:

```bash
taskman specified <id> --result "..." --done-when "..."
```

The task then waits for a human decision before implementation begins:

```bash
taskman specification approved <id> --comment "Looks good"
taskman specification rejected <id> --reason "Clarify the error behavior"
```

Tasks created with both `--specification` and `--done-when` enter `implement`
directly; those creation-time fields are treated as already approved.

After implementation, report that the change is ready for checks:

```bash
taskman implemented <id>
taskman verified <id> --check test=ok --check lint=ok
```

Failed checks enter `fix_verification_failure`. The failed check names and output identify
whether the work came from tests, lint, or another verifier. The agent fixes that failure and
reports another `verified` result. Verification failures have no fixed retry limit.

### Automated review: one fix round

An independent sub-agent reviews verified code before it is committed:

```text
automated_review
   ├─ approved → commit
   └─ rejected
          ↓
fix_automated_review_findings
          ↓
       verify
          ↓
automated_review again, using another sub-agent
   ├─ approved → commit
   └─ rejected → blocked
```

Report the verdict with:

```bash
taskman automated-review approved <id>

taskman automated-review rejected <id> \
  --finding internal/http/limiter.go="Limiter is shared across clients"
```

Both attempts use the same `automated_review` state. The review count is used only to enforce the
limit: one findings-fix round is allowed, and a second rejection blocks the task.

### Human review: unlimited fix rounds

After automated approval, the agent creates a Git commit and reports it:

```bash
taskman committed <id>
```

Taskman reads the commit directly from Git, then waits for explicit human review:

```text
human_review
   ├─ approved → merge
   └─ rejected
          ↓
fix_human_review_findings
          ↓
       verify
          ↓
     new commit
          ↓
human_review again
```

The human can run the review command or explicitly tell the agent to run it:

```bash
taskman review approved <id> --comment "Looks good"
taskman review rejected <id> --reason "Rate-limit headers are missing"
```

Human-review fixes do not return to automated review. This loop can repeat as many times as
needed. Once approved, the caller merges and reports completion:

```bash
taskman merged <id>
```

## Variants

`--trunk` works on the current branch and skips merge:

```bash
taskman create --trunk --definition "Fix a typo"
```

`--auto-approve` removes human review, but automated review still runs. After `committed`, an
isolated task goes to `merge`; a trunk task completes immediately.

## Blocking and abandonment

An agent that cannot continue safely can block a task:

```bash
taskman escalated <id> \
  --stage implementation \
  --reason "Required API behavior is ambiguous"
```

Taskman records the suspended state. There is currently no resume command; a blocked task waits
or can be abandoned:

```bash
taskman abandoned <id> --reason "Feature is no longer required"
```

`completed` and `abandoned` are terminal states.

## Storage

Tasks are YAML files under `.tasks/`:

```yaml
task:
  id: 0199...
  state: automated_review
  definition: Add per-client rate limiting
```

Agents should use `get`, `list`, and `next` instead of editing YAML. Updates use per-task locks
and atomic file replacement.

## Architecture

Taskman uses functional core, imperative shell:

```text
CLI or MCP request
        ↓
command shell: validate input and obtain external facts
        ↓
typed event
        ↓
task.Apply(current, event) → updated task
        ↓
locked, atomic persistence
```

`internal/task` is pure. It owns the task model, compiled workflow, typed events, reducers,
routes, validation, and instructions. It performs no filesystem, Git, clock, CLI, MCP, or prompt
rendering work.

Imperative adapters live in `internal/commands`, `internal/taskstore`, and `internal/gitclient`.
CLI and MCP frontends call the same command-slice functions.

```bash
taskman --json next <id>  # structured CLI output
taskman mcp               # MCP/stdio server
```

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
