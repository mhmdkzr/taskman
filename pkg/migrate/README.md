# `pkg/migrate`

Database migration runner built on `golang-migrate/migrate`. Applies SQL
migrations from an embedded filesystem (`embed.FS`) against a PostgreSQL
connection.

## API

- `Migrate(ctx, db, migrationsFS embed.FS) error` — runs all pending migrations
  from the given embedded FS in the default schema (no `CREATE SCHEMA`, uses
  PostgreSQL `public` via `schema = ""`). Shorthand for
  `MigrateSchema` with `sourceDir = "."` and `schema = ""`.
- `MigrateSchema(ctx, db, migrationsFS, sourceDir, schema string) error` —
  runs migrations from `sourceDir` inside the given PostgreSQL schema. When
  `schema` is non-empty, the schema is created first if missing;
  golang-migrate's `schema_migrations` table lives inside that schema. An empty
  `schema` uses the default `public` schema.

Both functions ping the database before starting, use the `iofs` source driver,
apply all pending migrations with `Up()` (a no-change result is not an error),
and close the migrate instance on return, joining close errors into the result.

## Usage

Migrations are embedded in `migrations.GetMigrationsFS()`; testdb's
`SetupCorePostgres` applies them automatically. Direct use:

```go
db := testdb.SetupCorePostgres(t, "myslice")
if err := migrate.MigrateSchema(ctx, db, otherFS, "migrations/other", "other"); err != nil {
	t.Fatalf("migrate: %v", err)
}
```

## Test fixture

`fixtures/001_create.up.sql` is a synthetic fixture embedded only by
`migrate_integration_test.go`; it is not part of the real migration set,
which lives in `migrations/`.
