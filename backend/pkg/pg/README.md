# `pkg/pg`

PostgreSQL connection helper. `Config` struct with `DSN()` method builds a connection string from component fields; `Open` opens and pings the database using the `lib/pq` driver.
