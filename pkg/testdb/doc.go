// Package testdb provides PostgreSQL test database lifecycle management using Testcontainers.
//
// SetupPostgres returns a *sql.DB backed by a fresh database forked from a
// single shared postgres container reused by name across test processes.
// Migrations run once against a template database; every test database is
// created via CREATE DATABASE ... TEMPLATE, so tests get full isolation without
// per-test container startup or migration runs. StartShared pre-warms the shared
// server from a TestMain, and SetServer points testdb at an existing migrated
// server instead of a container.
package testdb
