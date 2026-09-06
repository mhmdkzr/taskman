# notes_delete

Deletes the named note belonging to the current session. SQLite foreign keys cascade deletion to all links.

```json
{"name":"release-plan"}
```

Returns `{"deleted":true,"name":"release-plan"}` or an error when the note is missing.
