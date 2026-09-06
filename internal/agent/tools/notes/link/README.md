# notes_link

Creates or updates a link between two existing notes in the current session. Link IDs are stored in canonical order and repeated links update their relationship.

```json
{"from":"release-plan","to":"launch-checklist","relationship":"depends on"}
```

Returns the two note names and relationship.
