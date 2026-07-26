package middleware

import (
	"net/http"
	"slices"
)

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares to a handler in the order they are provided.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for _, v := range slices.Backward(mws) {
		h = v(h)
	}
	return h
}
