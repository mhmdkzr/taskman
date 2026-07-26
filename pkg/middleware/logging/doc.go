// Package logging provides HTTP middleware that logs a structured slog entry for every
// completed request (method, path, duration, status, client IP) with level based on
// response status code (Info for 2xx/3xx, Warn for 4xx, Error for 5xx).
package logging
