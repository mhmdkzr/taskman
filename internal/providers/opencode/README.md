# OpenCode Provider

The `opencode` package is the client for the OpenCode Zen API of the app's
configured provider (`PROVIDER_BASE_URL`, `PROVIDER_API_KEY_OPENCODE`). It
exposes the OpenCode Go plan usage status: how much of each quota window
(rolling ~5h, weekly, monthly) is used and when it resets.

## HTTP API

`GET /providers/opencode/usage` proxies `GET {PROVIDER_BASE_URL}/usage`
(authenticated with `PROVIDER_API_KEY_OPENCODE` as a Bearer token) and
responds with the normalized usage:

```json
{
  "rolling":  {"status": "ok", "percent": 2, "resets_at": "2026-09-07T12:12:30.464Z"},
  "weekly":   {"status": "ok", "percent": 0, "resets_at": "2026-09-14T00:00:00.464Z"},
  "monthly":  {"status": "ok", "percent": 50, "resets_at": "2026-09-30T17:15:56.464Z"}
}
```

- `status`: `ok` (quota left) or `rate-limited` (window exhausted, `percent`
  is 100)
- `percent`: share of the window's quota used (0-100); the share left is
  `100 - percent`
- `resets_at`: when the window resets

The upstream endpoint reports percentages only — raw spend and dollar limits
are not published by the API.

```sh
curl -sS http://localhost:8080/providers/opencode/usage
```

Errors: a missing provider configuration returns `500`; every upstream
failure (rejected key, missing subscription, network, unexpected response)
returns `502 Bad Gateway` with `{"error": "..."}`. `401` maps to
`ErrUnauthorized` and `403` to `ErrNoSubscription`.

## Upstream

```sh
# Query the upstream endpoint directly.
source .env
curl -sS -H "Authorization: Bearer $PROVIDER_API_KEY_OPENCODE" \
  "$PROVIDER_BASE_URL/usage"
```

The upstream wire format is
`usage.{rolling,weekly,monthly}.{status,percent,resetsAt}`, implemented in
anomalyco/opencode PR #16513: percentages are floored to integers and
`resetsAt` is an ISO timestamp.

Note: this covers the OpenCode Go subscription. The pay-as-you-go Zen
endpoint (`https://opencode.ai/zen/v1`) has no balance API yet
(anomalyco/opencode#10448).
