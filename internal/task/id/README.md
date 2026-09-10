# `internal/taskid`

Generates UUID-v7 task IDs and their readable title slugs. ID generation is
kept outside `internal/task` because it depends on randomness rather than the
pure workflow input.
