# taskman

**A deterministic task lifecycle manager for coding agents.**

Taskman gives coding agents a structured protocol for carrying a software task from definition to completion.

An agent does the actual work: inspecting the repository, writing code, running checks, reviewing changes, committing, and merging.

Taskman owns the lifecycle around that work. It keeps the authoritative task state, validates transitions, and at each state, it tells the agent what should happen next.

```text
definition
    ↓
specification
    ↓
implementation
    ↓
verification
    ↓
review
    ↓
merge
```

The main interaction is:

```bash
taskman next <id>
```

Taskman responds with a natural-language instruction to agent, describing the next valid step.

The agent follows it, reports the outcome with the indicated command to taskman, and calls `next` again, until the task is complete.

```text
agent runs taskman next
          ↓
taskman responds with instruction
          ↓
agent does the work
          ↓
agent reports outcome to taskman
          ↓
agent runs taskman next
          ↓
         ...
          ↓
        done
```

Taskman is designed to be operated **by agents**. It can be used as a CLI or as a MCP server (stdio). Humans typically interact with taskman indirectly through an agent.

> **Status:** alpha. Expect breaking changes and potential bugs.

## Why taskman?

When you use tasks as unit of work, and tasks are well defined and specific enough, you no longer need to become the agent operator, you can let taskman handle it. Taskman gives the agents a clear, structured workflow for performing tasks, while keeping track of the task state and progress.

Taskman does not try to make decisions that belong to the coding agent. It is deliberately deterministic in its workflow.

## Agent-first interface

Taskman can be used as a CLI tool, or as a MCP server (stdio). Both interfaces have same commands available.

Commands such as:

```bash
taskman next <id>
```

return concise natural-language guidance containing the context and command needed to continue the task, similar to how you would do it, if you were to guide the agent.

## Getting an agent started

Running `taskman skill` will print out the SKILL.md contents for taskman, put it besides your other agent skill files. The skill teaches an agent how to drive taskman correctly.

## Creating a task

A task starts with a definition:

```bash
taskman create \
  --definition "Add per-client rate limiting to the HTTP API" \
  --specification "Use a token-bucket limiter keyed by client ID" \
  --done-when "Requests above the configured rate receive HTTP 429"
```

By default taskman creates an isolated Git branch and worktree for the task. You can use the `trunk` mode to skip the branch/worktree creation and use the current directory instead.

The returned task ID can then be given to an agent:

```text
Use taskman to perform task <id>. Follow its instructions.
```

From there, the agent can drive the lifecycle with:

```bash
taskman next <id>
```

## The task lifecycle

Taskman models six stages:

```text
definition → specification → implementation → verification → review → merge
```

Each transition is explicit.

Commands that are invalid for the current stage are rejected.

### Definition

Definition determines what the task is, without specifying how it should be done.

`create` records the task and establishes its Git environment.

```bash
taskman create \
  --definition "Add rate limiting"
```

Optional metadata can be attached:

```bash
taskman create \
  --definition "Fix token refresh race" \
  --label service=auth \
  --label priority=high \
  --reference internal/auth/token.go
```

### Specification

If the task does not already have a specification, `next` asks an agent to produce one.

The result is recorded with:

```bash
taskman specify <id> \
  --result "Use a token-bucket limiter keyed by client ID..." \
  --done-when "Requests above the configured rate receive HTTP 429..."
```

Specification describes how the task should be done.

`done_when` describes the conditions under which the task can be considered correct.

### Implementation

The agent performs the implementation in the task's worktree.

When that implementation attempt is complete:

```bash
taskman implement <id>
```

This does not claim the implementation is correct. It only records that it is ready for verification.

### Verification

The agent runs the required checks and reports their outcomes:

```bash
taskman verify <id> \
  --check tests=ok \
  --check lint=ok
```

Failures can include diagnostic output:

```bash
taskman verify <id> \
  --check tests=error \
  --output "TestRateLimiter: expected 429, got 200"
```

Verification attempts are appended to the task history.

If verification fails, the lifecycle returns to implementation.

### Automated review

After verification succeeds, an agent reviews the implementation. Taskman asks the agent to use a sub-agent for the review.

Its verdict is recorded with:

```bash
taskman review record <id> --approved true
```

or:

```bash
taskman review record <id> \
  --approved false \
  --finding internal/http/limiter.go="Limiter is shared across clients"
```

A rejected review returns the task to implementation, if rejected again, it won't be retried.

Review history is retained rather than overwritten.

### Commit and human review

After automated review succeeds, the agent creates the code commit.

Taskman then reads the commit from Git:

```bash
taskman commit <id>
```

Unless the task was created with `--auto-approve`, the next stage requires human review. An agent can present the change for review and then record the human's decision:

```bash
taskman review approve <id> \
  --comment "Looks good"
```

or:

```bash
taskman review reject <id> \
  --reason "Rate-limit headers are missing"
```

A rejection starts another implementation and verification cycle.

### Merge

Once review is complete, the agent merges the branch and reports the result:

```bash
taskman merge <id>
```

The task then becomes `completed`. If `trunk` mode is used, this step is skipped since there's nothing to merge.

## Blocking a task

Sometimes an agent cannot continue safely.

For example:

* the specification is ambiguous;
* required information is unavailable;
* proceeding would require a decision from a human.

The agent can block the task:

```bash
taskman escalate <id> \
  --stage implementation \
  --reason "Required API behavior is ambiguous"
```

`taskman next <id>` will then report that the task is waiting rather than instructing another agent to continue blindly.

Once the issue is resolved, the task can be updated and resumed through the normal lifecycle.

## Abandoning a task

If the work should be stopped permanently:

```bash
taskman abandon <id> \
  --reason "Feature is no longer required"
```

The task becomes `failed`.

## Git integration

Git is part of taskman's execution model.

By default, each task gets an isolated branch and worktree under:

```text
.worktrees/
```

This allows multiple agents or tasks to operate independently without sharing a mutable checkout.

Taskman also verifies Git facts directly where possible.

For example, after an agent commits an implementation, taskman reads the commit from the worktree rather than relying solely on a caller-provided hash.

The responsibility split is roughly:

```text
agent                                  taskman
─────                                  ───────

understand the task
inspect code
edit files
run checks
review changes
create code commit          ───────►  inspect repository state
perform merge               ───────►  record outcome

                              ◄──────  validate lifecycle state
                              ◄──────  provide next instruction
                              ◄──────  persist task state
```

Taskman creates a Git commit itself only for terminal task bookkeeping, as the task files are tracked by Git, the task file continues changing through review and merge after the implementation commit already exists.

## Isolated worktrees

By default, creating a task requires a clean repository and creates an isolated worktree and branch.

The worktree path is stored with the task so an agent can operate in the correct checkout.

For workflows where isolation is unnecessary:

```bash
taskman create \
  --trunk \
  --definition "Fix typo in error message"
```

A trunk task operates directly on the current branch and has no separate merge stage.

## Task state

Each task is persisted as a YAML file:

```text
.tasks/<uuid-v7>_<slug>.yaml
```

It contains the authoritative state needed to continue the workflow, including:

* task definition;
* specification;
* acceptance criteria;
* lifecycle state;
* per-stage progress;
* labels and repository references;
* Git branch and worktree;
* recorded implementation commit;
* verification attempts;
* automated review rounds;
* human review decisions;
* blocking and failure information.

These files are implementation state, not the normal interface an agent needs to use.

Agents should interact through taskman's commands, and avoid editing these files directly. Taskman handles concurrent mutation with per-task lock files. Different processes can therefore work with different tasks in the same repository safely.

## Human involvement

Human role is in task definition and review. It is important to understand that not everything can be expressed in tasks and done in this workflow, sometimes you need to be more involved, taskman won't help you in those cases. In other cases where the work is well-defined and can be trusted with an agent to do it, taskman can help you automate the workflow: write the tasks, let taskman drive the agent to do the work, monitor the progress and review the results if necessary.

## Auto-approval

If a task does not require a separate human approval step, you can create the task with `--auto-approve` to remove the human review gate:

```bash
taskman create \
  --auto-approve \
  --definition "..."
```

## Programmatic interface

The default text output is optimized for agent consumption. Taskman also supports a JSON output mode:

```bash
taskman --json next <id>
```

JSON is intended for programmatic access that require structured output. JSON output is fuller than the normal text response. 

## MCP

Taskman also exposes its operations over MCP/stdio:

```bash
taskman mcp
```

This allows compatible agent runtimes to invoke taskman as tools rather than shell commands.

The CLI and MCP interfaces use the same underlying task model and lifecycle rules.

## Command reference

| Command               | Purpose                                             |
| --------------------- | --------------------------------------------------- |
| `list`                | List tasks                                          |
| `get <id>`            | Inspect a task                                      |
| `next <id>`           | Tell an agent what should happen next               |
| `create`              | Create a task and its worktree                      |
| `update <id>`         | Update task metadata or settings                    |
| `specify <id>`        | Record a specification and acceptance criteria      |
| `implement <id>`      | Record completion of an implementation attempt      |
| `verify <id>`         | Record a verification attempt                       |
| `review record <id>`  | Record an automated review verdict                  |
| `review approve <id>` | Record human approval                               |
| `review reject <id>`  | Record human rejection                              |
| `commit <id>`         | Record the implementation commit from Git           |
| `escalate <id>`       | Block a task                                        |
| `merge <id>`          | Record completion of the merge                      |
| `abandon <id>`        | Permanently fail a task                             |
| `delete <id>`         | Delete a task                                       |
| `prune`               | Remove completed task files                         |
| `skill`               | Print taskman's agent-facing operating instructions |
| `mcp`                 | Serve taskman over MCP/stdio                        |

Every command supports `--help`.

## Global options

| Flag              | Default      | Purpose                                            |
| ----------------- | ------------ | -------------------------------------------------- |
| `--git-dir`       | `.`          | Repository root                                    |
| `--tasks-dir`     | `.tasks`     | Task-state directory                               |
| `--worktrees-dir` | `.worktrees` | Worktree directory                                 |
| `--json`          | off          | Emit the full machine-readable JSON representation |
| `--log-level`     | `info`       | `debug`, `info`, `warn`, or `error`                |
| `--log-format`    | `text`       | `text` or `json`                                   |

## Installation

Requires Go 1.27+ and `git` on `PATH`.

Install directly:

```bash
go install github.com/mhmdkzr/taskman@latest
```

Or from a checkout:

```bash
go install .
```

## Design principles

### Agents are the users

Taskman is built to be invoked by coding agents.

Humans express intent to those agents and may participate in approval decisions, but they should not need to manually drive the task lifecycle.

### Natural language is the agent interface

LLMs already understand concise textual instructions.

Taskman therefore emits natural-language guidance by default instead of requiring agents to consume a verbose structured representation.

Structured JSON exists for ordinary software integrations.

### Task state lives outside the agent

The current lifecycle state does not depend on a particular conversation, model invocation, or agent process.

A later invocation—or a different agent—can inspect the task and continue from the same explicit state.

### Taskman controls transitions, not implementation

Taskman determines whether a lifecycle transition is valid.

It does not determine how the implementation should be written.

That reasoning belongs to the agent.

### Git is authoritative where possible

Agents report outcomes, but taskman verifies facts directly from Git when it can.

### Failure is part of the lifecycle

Verification failures and review rejections are expected transitions, not exceptional corruption of the workflow.

### History is retained

Verification attempts and review rounds are append-only, so later agents can see the progression of the task.

### The interface should stay small

An agent should not need to understand taskman's internal state machine in order to use it.

The preferred interaction is:

```bash
taskman next <id>
```

follow the instruction, report the result, repeat.

## Repository layout

The codebase follows a vertical slice architecture. 

```text
main.go
    thin executable entrypoint

internal/task
    task domain model
    persistence
    transition primitives
    Git integration

internal/commands
    command slices
    CLI frontends
    MCP frontends
    agent instructions

internal/cli
    CLI assembly

internal/mcp
    MCP server assembly

internal/utils
    shared CLI plumbing
```

See the READMEs under `internal/` for implementation details.

## Development

Build:

```bash
make build
```

Run tests:

```bash
make test
```

Run static analysis:

```bash
make lint
```

Format the repository:

```bash
make fmt
```

## License

MIT
