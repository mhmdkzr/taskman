# apply_patch

`apply_patch` applies OpenCode's multi-file patch format. It accepts one
`patchText` value and supports adding, updating, deleting, and moving files.
All patch hunks are parsed and validated before any file is written.

Example:

```text
*** Begin Patch
*** Update File: config.go
@@
-old value
+new value
*** End Patch
```
