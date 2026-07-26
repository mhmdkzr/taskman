// Package app provides the core application container (App) aggregating Deps (PostgreSQL,
// NATS, JetStream, Temporal), Cfg (application config from internal/config), and Mux
// (HTTP router with Handle helper). This is the primary dependency injection hub.
package app
