// Package auditlog provides HTTP middleware that captures full request/response details
// (method, path, headers, bodies, status, timing) and publishes them as JetStream events
// to the API_AUDIT stream. Also provides a durable consumer that persists events to PostgreSQL.
package auditlog
