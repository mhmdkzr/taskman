# prune

`taskman prune` deletes the task files (`.tasks/*.yaml`) of every task in the
`completed` state, in one shot. It is the bulk counterpart to `delete <id>`,
which removes a single task regardless of its state.

## What it does

- Enumerates the task files under the tasks directory.
- Removes the file of each task whose state is `completed`.
- Leaves tasks in every other fine-grained workflow state untouched.
- Touches nothing but task files: no soft-delete, no git worktree or branch
  cleanup, no commits. Git history covers "undo".

## Invocation

```
taskman prune [--dry-run]
```

- `--dry-run` lists the completed tasks that would be deleted without
  actually removing anything.
- `--json` returns the result as a JSON envelope `{deleted, count}` instead
  of the human-readable listing.

## Behavior

- Prints each pruned task id (or, with `--dry-run`, each id that would be
  pruned) followed by a count.
- With no completed tasks, prints `no completed tasks` and exits successfully.
- A missing tasks directory is not an error - nothing is pruned.
- This is a human housekeeping action; never invoke it while driving a task.
