# delete

`taskman delete <id>` removes a single task's file (`.tasks/<id>.yaml`) and
lock artifact outright, regardless of its state. It is the single-task
counterpart to `prune`, which removes every `completed` task at once.

## What it does

- Removes the task file and lock artifact for the given id, whatever state it
  is in.
- Performs no soft-delete, git worktree or branch cleanup, or commits. Git
  history covers "undo" for the tracked task file.

## Invocation

```
taskman delete <id>
```

## Behavior

- Prints `deleted task <id>` on success.
- Fails with `task not found` (exit 1) if the id has no task file.
- Fails with a usage error (exit 2) if no id is given.
- This is a human housekeeping action; never invoke it while driving a task.
