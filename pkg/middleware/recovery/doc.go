// Package recovery provides HTTP middleware that recovers from panics in downstream
// handlers, logs them, and returns a 500 response instead of crashing the server.
package recovery
