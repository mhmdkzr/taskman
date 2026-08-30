package bff

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mhmdkzr/app/pkg/jsonresp"
)

// Handler exposes browser-facing BFF endpoints.
type Handler struct{ service *Service }

// NewHandler creates an enabled BFF handler.
func NewHandler(service *Service) Handler { return Handler{service: service} }

// Login is the browser-navigation sign-in endpoint.
func (h Handler) Login(w http.ResponseWriter, r *http.Request) { h.login(w, r) }

// Callback completes the authorization-code flow.
func (h Handler) Callback(w http.ResponseWriter, r *http.Request) { h.callback(w, r) }

// Logout destroys the local app session.
func (h Handler) Logout(w http.ResponseWriter, r *http.Request) { h.logout(w, r) }

// Me returns the authenticated app-owned identity.
func (h Handler) Me(w http.ResponseWriter, r *http.Request) { h.me(w, r) }

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	state, err := randomValue()
	if err != nil {
		jsonresp.WriteHTTPError(w, 500, err)
		return
	}
	verifier, err := randomValue()
	if err != nil {
		jsonresp.WriteHTTPError(w, 500, err)
		return
	}
	nonce, err := randomValue()
	if err != nil {
		jsonresp.WriteHTTPError(w, 500, err)
		return
	}
	h.service.transactions.Put(r.Context(), stateKey, state)
	h.service.transactions.Put(r.Context(), verifierKey, verifier)
	h.service.transactions.Put(r.Context(), nonceKey, nonce)
	h.service.transactions.Put(r.Context(), returnToKey, safeReturnTo(r.URL.Query().Get("return_to")))
	h.service.transactions.SetDeadline(r.Context(), time.Now().Add(10*time.Minute))
	http.Redirect(w, r, h.service.authorizationURL(state, verifier, nonce), http.StatusFound)
}

func (h Handler) callback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("error") != "" {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, errInvalidCallback)
		return
	}
	returnTo, err := h.service.CompleteAuthorizationCallback(
		r.Context(), r.URL.Query().Get("code"), r.URL.Query().Get("state"),
	)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	// Re-applied even though CompleteAuthorizationCallback already sanitizes
	// returnTo: it crosses a package boundary, so treat it as untrusted here too.
	safeRedirect := safeReturnTo(returnTo)
	//nolint:gosec // safeRedirect is validated by safeReturnTo to be a local path
	http.Redirect(w, r, safeRedirect, http.StatusFound)
}

// httpStatusForError maps CompleteAuthorizationCallback's error outcomes to
// HTTP status codes: a malformed/mismatched callback is the caller's fault,
// a failed introspection means the token ZITADEL issued was rejected, and
// anything else (code exchange, provisioning, session storage) is this
// service failing to complete a callback ZITADEL considers valid.
func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, errInvalidCallback):
		return http.StatusBadRequest
	case errors.Is(err, errUnauthenticated):
		return http.StatusUnauthorized
	default:
		return http.StatusBadGateway
	}
}

func (h Handler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
		jsonresp.WriteHTTPError(w, http.StatusForbidden, errors.New("csrf protection: missing X-Requested-With header"))
		return
	}
	if err := h.service.sessions.Destroy(r.Context()); err != nil {
		jsonresp.WriteHTTPError(w, 500, fmt.Errorf("destroy session: %w", err))
		return
	}
	if h.service.discovery != nil {
		end, err := url.Parse(h.service.discovery.EndSessionEndpoint)
		if err == nil && end.String() != "" {
			q := end.Query()
			q.Set("client_id", h.service.cfg.ClientID)
			q.Set("post_logout_redirect_uri", h.service.cfg.PostLogoutRedirectURL)
			end.RawQuery = q.Encode()
			jsonresp.WriteJSON(w, http.StatusOK, struct {
				RedirectURL string `json:"redirect_url"`
			}{RedirectURL: end.String()})
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) me(w http.ResponseWriter, r *http.Request) {
	principal, err := h.service.principal(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusUnauthorized, err)
		return
	}
	jsonresp.WriteJSON(w, http.StatusOK, struct {
		ID string `json:"id"`
	}{ID: principal.String()})
}
