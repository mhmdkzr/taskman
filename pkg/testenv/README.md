# `pkg/testenv`

Test environment helpers: loading `.env` files, reading environment variables
with defaults, and skipping integration tests unless explicitly enabled.

## Environment resolution

`EnvOrDefault(key, fallback string) string` resolves a value in priority order:

1. The process environment variable (if non-empty after trimming whitespace).
2. The value from the nearest `.env` file (see below).
3. The provided fallback.

The `.env` file is found by walking up from the current working directory to
the root. It is loaded once per process; blank lines and `#` comments are
skipped, an optional `export ` prefix is stripped, values are trimmed of
surrounding quotes (`"` or `'`), and malformed lines without `=` are ignored.

`DotEnvValue(key string) string` reads a key directly from the loaded `.env`
file (empty string if absent).

## Test gating

Integration tests are opt-in via environment variables; each helper skips the
test when its variable is unset or empty:

| Helper | Variable |
|---|---|
| `SkipIfDBTestsDisabled(tb)` | `RUN_DB_TESTS=1` |
| `SkipIfNetworkTestsDisabled(tb)` | `RUN_NETWORK_TESTS=1` |
| `SkipIfE2ETestsDisabled(tb)` | `RUN_E2E_TESTS=1` |

## Usage

```go
func TestDBThing(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	dsn := testenv.EnvOrDefault("TEST_DATABASE_URL", defaultDSN)
	// ...
}
```
