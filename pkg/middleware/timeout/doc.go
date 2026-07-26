// Package timeout provides HTTP middleware that applies context.WithTimeout to every request,
// ensuring long-running handlers are canceled and cannot hang the server.
package timeout
