# Query Tool

The `query` tool runs one read-only SQL statement against the agent database.
It accepts `SELECT`, `WITH`, and `EXPLAIN` statements only, uses the store's
read-only SQLite connection, limits results to 1,000 rows, and truncates long
cells to keep tool output bounded.

```json
{"query":"SELECT name FROM agents ORDER BY name"}
```
