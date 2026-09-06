# Todo Tool

`todo` replaces the active agent session's todo list. Each todo has `content`, `status`, and `priority`; status is one of `pending`, `in_progress`, `completed`, or `cancelled`, and priority is `high`, `medium`, or `low`.

The tool result includes the complete submitted list. This lets session history render each `todo` call as a separate todo set, while the database retains only the latest active list for the session.

Example input:

```json
{
  "todos": [
    {
      "content": "Run tests",
      "status": "in_progress",
      "priority": "high"
    }
  ]
}
```

Example result:

```json
{
  "todos": [
    {
      "content": "Run tests",
      "status": "in_progress",
      "priority": "high"
    }
  ]
}
```
