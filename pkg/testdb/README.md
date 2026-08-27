# `pkg/testdb`

PostgreSQL test database lifecycle management using Testcontainers. Provides
fully isolated, migrated databases for DB-backed integration tests without
per-test container startup or migration runs.

## How it behaves

- A single PostgreSQL container (`postgres:18-trixie`, named
  `testdb-pg`) is reused by name across all test processes in a
  `go test` run via Testcontainers reuse.
- Migrations run once against a template database (`test_template`).
  A cluster-wide Postgres advisory lock serializes that one-time migration
  across processes.
- Each `SetupPostgres(t, suffix)` call forks a fresh database from the
  template via `CREATE DATABASE ... TEMPLATE`, so tests never share state and
  get a fully migrated database every time.
- The returned `*sql.DB` is pinged before use and closed automatically via
  `t.Cleanup`.
- The shared container persists between runs: reuse-by-name means subsequent
  processes reconnect to the same server, and nothing removes it automatically.

## Stale template

Migrations only ever run against the shared container's template database
while that template is still unmigrated; a reused template is observed and
skipped as-is (`migrateTemplateIfNeeded` in `postgres.go`). Because the
container persists between runs, **newly added migrations are invisible to
your tests until the shared container — and with it the template — is reset**:

```sh
docker rm -f testdb-pg
```

The next test run starts a fresh container and migrates its template from
scratch, picking up the new migrations.

## API

- `SetupPostgres(t *testing.T, suffix string) *sql.DB` — the entry point
  used by every DB-backed test. Lazily starts (or reuses) the shared container,
  ensures the migrated template exists, and returns a new forked database.
  `suffix` distinguishes the caller package in the generated database name.
- `StartShared() error` — pre-warms the shared server (container + migrated
  template) so the one-time cost is paid up front rather than by the first
  `SetupPostgres` call. Idempotent; call from a package `TestMain`.
- `SetServer(dsn string)` — point testdb at an existing, already-migrated
  server instead of a container. `dsn` is the server base without a database
  name, e.g. `postgres://postgres:postgres@host:port/`. Must be called before
  any `SetupPostgres` call.

## Usage

```go
func TestMain(m *testing.M) {
	if err := testdb.StartShared(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestDBSomething(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	db := testdb.SetupPostgres(t, "myslice")
	// seed fixtures and exercise slice logic against db
}
```
