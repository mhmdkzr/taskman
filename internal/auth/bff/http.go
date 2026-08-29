package bff

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mhmdkzr/app/internal/identity/provision"
	"github.com/mhmdkzr/app/pkg/jsonresp"
)

var errInvalidCallback = errors.New("invalid authentication callback")

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
	expectedState := h.service.transactions.GetString(r.Context(), stateKey)
	callbackState := r.URL.Query().Get("state")
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" || expectedState == "" ||
		subtle.ConstantTimeCompare([]byte(callbackState), []byte(expectedState)) != 1 {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, errInvalidCallback)
		return
	}
	verifier := h.service.transactions.GetString(r.Context(), verifierKey)
	returnTo := safeReturnTo(h.service.transactions.GetString(r.Context(), returnToKey))
	if err := h.service.transactions.Destroy(r.Context()); err != nil {
		jsonresp.WriteHTTPError(w, http.StatusInternalServerError, fmt.Errorf("destroy auth transaction: %w", err))
		return
	}
	accessToken, refreshToken, expiry, err := h.service.exchange(r.Context(), r.URL.Query().Get("code"), verifier)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadGateway, err)
		return
	}
	subject, err := h.service.introspect(r.Context(), accessToken)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusUnauthorized, err)
		return
	}
	userID, err := provision.FindOrCreate(r.Context(), h.service.db, subject)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	if err := h.service.sessions.RenewToken(r.Context()); err != nil {
		jsonresp.WriteHTTPError(w, 500, fmt.Errorf("renew session: %w", err))
		return
	}
	h.service.sessions.Put(r.Context(), accessTokenKey, accessToken)
	h.service.sessions.Put(r.Context(), refreshTokenKey, refreshToken)
	h.service.sessions.Put(r.Context(), expiryKey, expiry)
	h.service.sessions.Put(r.Context(), userIDKey, userID.String())
	http.Redirect(w, r, returnTo, http.StatusFound)
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
