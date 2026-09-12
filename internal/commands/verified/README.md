# verified

Records one verification attempt and derives pass/fail from the reported checks.

`verified` appends `VerificationPassed` unless any reported check is `error`, in which case it
appends `VerificationFailed` - there is no way to report a failing check as a pass. The reported
checks must match exactly what `implemented` required: every required check present, no others.

A pass routes to `automated_review` (if required) or `commit`; a failure loops back through
`fix_verification_failure` with no retry limit. A task that declares no verification checks never
enters `verify`, so it never reports one.

## Request

| Field | Flag | Required |
|---|---|---|
| `ID` | `--id` | yes |
| `Checks` | `--unit` / `--integration` / `--end-to-end` / `--linters` (`ok` or `error`) | at least one |
| `Output` | `--output` | no |

## CLI

```bash
taskman verified --id <id> --unit ok --linters error --output "2 lint issues"
```

## MCP

`task_verified`.
