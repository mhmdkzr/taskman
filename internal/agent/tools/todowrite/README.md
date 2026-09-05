# Todowrite Tool

`todowrite` replaces the active agent session's todo list. Each todo has `content`, `status`, and `priority`; status is one of `pending`, `in_progress`, `completed`, or `cancelled`, and priority is `high`, `medium`, or `low`.

Example input:

```json
{"todos":[{"content":"Run tests","status":"in_progress","priority":"high"}]}
```
