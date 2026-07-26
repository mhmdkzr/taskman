// Package process bootstraps and starts the application: loads config, connects
// PostgreSQL/NATS/JetStream/Temporal, initializes logger, runs migrations, starts
// the HTTP server (with middleware chain) and Temporal worker.
package process
