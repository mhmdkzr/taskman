# `health/get`

Health query slice that exposes a lightweight liveness endpoint.

## Route

- `GET /health`

## Behavior

- Returns `200 OK` with JSON payload `{"status":"ok"}`; this handler never inspects dependencies.
- Includes `"commit"` with the git revision the binary was built from, when the build recorded VCS info (omitted otherwise via `omitempty`).
- Does not perform deep dependency checks; use `GET /ready` for dependency probes.

## Example

```bash
curl -sS http://127.0.0.1:8090/health
```

Example response:

```json
{
  "status": "ok",
  "commit": "7034b82c"
}
```
