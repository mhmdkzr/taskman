# `health/get`

Health query slice that exposes a lightweight liveness endpoint for core.

## Route

- `GET /health`

## Behavior

- Returns `200 OK` with JSON payload `{"status":"ok"}`.
- Does not perform deep dependency checks; this endpoint is for process liveness.

## Example

```bash
curl -sS http://127.0.0.1:8090/health
```

Example response:

```json
{"status":"ok"}
```
