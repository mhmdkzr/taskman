// Package resp provides helpers for writing HTTP responses.
package resp

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

// ResponseError is the standard JSON error body returned on non-successful HTTP responses.
type ResponseError struct {
	Error string `json:"error"`
}

// WriteJSON marshals v as JSON and writes it with the given status code.
// On marshal failure it returns a 500 with a static error body.
func WriteJSON[T any](w http.ResponseWriter, status int, v T) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		slog.Error("failed to encode json response", "status", status, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if _, writeErr := w.Write([]byte(`{"error":"failed to encode json response"}`)); writeErr != nil {
			slog.Error("failed to write json error response", "error", writeErr)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(buf.Bytes()); err != nil {
		slog.Error("failed to write json response", "status", status, "error", err)
	}
}

// WriteHTTPError writes err.Error() as a JSON error response with the given status.
func WriteHTTPError(w http.ResponseWriter, status int, err error) {
	WriteJSON(w, status, ResponseError{Error: err.Error()})
}
