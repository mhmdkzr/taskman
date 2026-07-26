// Package migrate runs SQL database schema migrations using golang-migrate/migrate.
// It accepts an embedded filesystem containing up/down migration pairs and applies
// pending migrations against a PostgreSQL connection.
package migrate
