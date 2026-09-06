# notes_search

Searches note names and bodies with SQLite FTS5 within the current session. `limit` defaults to 20 and is bounded to 100.

```json
{"query":"Friday","limit":10}
```

Returns matching note ids, names, bodies, and creation timestamps ordered by FTS relevance.
