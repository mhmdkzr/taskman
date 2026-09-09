# delete

`taskman delete <id>` removes a single task's file (`.tasks/<id>.yaml`)
outright, regardless of its state. It is the single-task counterpart to
`prune`, which removes every `completed` task at once.

## What it does

- Removes the task file for the given id, whatever state it is in.
- Touches nothing but that one file: no soft-delete, no git worktree or
  branch cleanup, no commits. Git history covers "undo".

## Invocation

```
taskman delete <id>
```

## Behavior

- Prints `deleted task <id>` on success.
- Fails with `task not found` (exit 1) if the id has no task file.
- Fails with a usage error (exit 2) if no id is given.
- This is a human housekeeping action; never invoke it while driving a task.
