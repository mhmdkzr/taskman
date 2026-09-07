package opencode

import (
	"errors"
	"net/http"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/pkg/jsonresp"
)

// usageHandler responds with the configured provider's OpenCode Go usage.
func usageHandler(a app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usage, err := FetchUsage(r.Context(), a.Cfg.Provider)
		if err != nil {
			jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
			return
		}
		jsonresp.WriteJSON(w, http.StatusOK, usage)
	}
}

// httpStatusForError maps usage errors to HTTP statuses. A missing provider
// configuration is a server-side defect (500); every other failure is
// upstream (rejected key, missing subscription, network, unexpected
// response) while the caller's request is fine, so it surfaces as Bad
// Gateway.
func httpStatusForError(err error) int {
	if errors.Is(err, ErrNotConfigured) {
		return http.StatusInternalServerError
	}
	return http.StatusBadGateway
}
