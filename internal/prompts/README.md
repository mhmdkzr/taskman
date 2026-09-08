# `internal/prompts`

Every natural-language template taskman renders - see `notes/design/design.md` §4. Templates are
plain Markdown files, embedded into the binary at compile time (`go:embed`, `prompts.go`) and
parsed once at startup - there is no runtime file lookup, no directory flag, and no way to edit a
prompt without rebuilding taskman. Each template has its own typed params struct and `Render()`
method, so a caller can't pass the wrong shape of data into the wrong template.

## Judgment-dispatch prompts

Rendered and handed to whatever agent a caller dispatches - the text a human would otherwise type
by hand:

| Template | Struct | Used for |
| --- | --- | --- |
| `specify.md` | `Specify` | Drafting `specification`/`done_when` from `definition` |
| `implement.md` | `Implement` | Writing the task's diff in its worktree |
| `fix.md` | `Fix` | Fixing a failed build check, a rejected automated review, or a human rejection (`Reason` carries whichever) |
| `review.md` | `Review` | The automated review round inside verification |
| `commit.md` | `Commit` | Drafting and making a commit for the current diff |

## Wrapping and mechanical-step templates

| Template | Struct | Used for |
| --- | --- | --- |
| `dispatch_wrapper.md` | `DispatchWrapper` | Wraps a judgment-dispatch prompt's body with the worktree/branch/report-back context common to every dispatch |
| `run_verify.md` | `RunVerify` | The `run` action for the first (mechanical) build check |
| `run_merge.md` | `RunMerge` | The `run` action for the merge step |

## `task next`'s wait/done messages

| Template | Struct | Used for |
| --- | --- | --- |
| `wait_human_review.md` | `WaitHumanReview` | Awaiting `task review approve`/`reject` |
| `wait_blocked.md` | `WaitBlocked` | A blocked task, awaiting human intervention |
| `done_merged.md` | `DoneMerged` | A completed task |
| `done_abandoned.md` | `DoneAbandoned` | An abandoned task |

## CLI default output

| Template | Struct | Used for |
| --- | --- | --- |
| `create_summary.md` | `CreateSummary` | `task create`'s default (non-JSON) output |
| `task_summary.md` | `TaskSummary` | Every other `Task`-returning command's default output |

## Adding a template

1. Add the `.md` file (picked up automatically by `//go:embed *.md`).
2. Add a `var xTmpl = parse("x.md")` and a params struct with a `Render()` method in `prompts.go`.
3. Use it from `internal/task` (usually `next.go`) or `internal/cli`.
