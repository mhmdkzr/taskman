# Notes Write Tool

The `notes_write` tool creates or updates a persistent note for the current
session. Note names are globally unique. Updating a note replaces its body and
all of its links; each link names an existing note and may include a
relationship description. The note and links are saved atomically.

```json
{"name":"release-plan","body":"Ship after the migration is verified.","links":[{"name":"migration","relationship":"depends on"}]}
```
