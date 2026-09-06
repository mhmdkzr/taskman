# History Tool

The `history` tool reads past conversation turns across agent sessions,
newest first. It uses the store's read-only SQLite connection, matches
`query` against a turn's prompt or reply text (case-insensitive substring),
optionally restricts to one session, limits results to 50 turns, and
truncates long prompts/replies to keep tool output bounded.

```json
{
  "query": "reporting bug"
}
```

```json
{
  "session_id": "0198f2b0-6f1a-7c33-9a2e-6b6f1a7c339a",
  "limit": 5
}
```
