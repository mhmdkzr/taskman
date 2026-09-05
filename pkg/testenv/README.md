# `pkg/testenv`

Test environment helpers: loading `.env` files, reading environment variables
with defaults, and skipping integration tests unless explicitly enabled.

## Environment resolution

`EnvOrDefault(key, fallback string) string` resolves a value in priority order:

1. The process environment variable (if non-empty after trimming whitespace).
2. The value from the nearest `.env` file (see below).
3. The provided fallback.

The `.env` file is found by walking up from the current working directory to
the root. It is loaded once per process; blank lines and lines whose first
non-whitespace character is `#` are skipped, an optional `export ` prefix is
stripped, values are trimmed of surrounding quotes (`"` or `'`), and malformed
lines without `=` are ignored.

Only full-line comments are recognized: a `#` that appears inline (e.g.
`KEY=value # note`) is kept as part of the value (`value # note`), so do not
use trailing comments in `.env` values.

`DotEnvValue(key string) string` reads a key directly from the loaded `.env`
file (empty string if absent).

## Test gating

Integration tests are opt-in via environment variables; each helper skips the
test unless its variable is exactly `1` (unset, empty, or non-truthy values such
as `0` or `false` all skip):

| Helper | Variable |
|---|---|
| `SkipIfDBTestsDisabled(tb)` | `RUN_DB_TESTS=1` |
| `SkipIfE2ETestsDisabled(tb)` | `RUN_E2E_TESTS=1` |

## Usage

```go
func TestDBThing(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	dsn := testenv.EnvOrDefault("TEST_DATABASE_URL", defaultDSN)
	// ...
}
```
