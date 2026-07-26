# `pkg/testdb`

Test database setup. Spins up a disposable PostgreSQL container via Testcontainers, runs core migrations, and returns a ready-to-use `*sql.DB` for integration tests.
