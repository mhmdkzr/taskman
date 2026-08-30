package loginui

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mhmdkzr/app/pkg/jsonresp"
)

// Handler exposes the browser-facing endpoints the custom Svelte login form
// calls to drive a ZITADEL session through username+password checks and
// finalize it against an OIDC authorization request.
type Handler struct{ service *Service }

// NewHandler creates a login UI handler.
func NewHandler(service *Service) Handler { return Handler{service: service} }

// CreateSession starts a login for a login name.
func (h Handler) CreateSession(w http.ResponseWriter, r *http.Request) { h.createSession(w, r) }

// CheckPassword verifies the password for the in-progress login.
func (h Handler) CheckPassword(w http.ResponseWriter, r *http.Request) { h.checkPassword(w, r) }

// CheckTOTP verifies a TOTP code for the in-progress login, when the
// account has TOTP enrolled (State.NeedsTOTP).
func (h Handler) CheckTOTP(w http.ResponseWriter, r *http.Request) { h.checkTOTP(w, r) }

// Finalize links a sufficient login to an OIDC authorization request.
func (h Handler) Finalize(w http.ResponseWriter, r *http.Request) { h.finalize(w, r) }

type createSessionRequestBody struct {
	LoginName string `json:"loginName"`
}

func createSessionRequestFromHTTP(r *http.Request) (createSessionRequestBody, error) {
	var body createSessionRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return createSessionRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}
	if body.LoginName == "" {
		return createSessionRequestBody{}, errors.New("loginName must not be empty")
	}
	return body, nil
}

func (h Handler) createSession(w http.ResponseWriter, r *http.Request) {
	body, err := createSessionRequestFromHTTP(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
		return
	}
	state, err := h.service.startWithLoginName(r.Context(), body.LoginName)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	jsonresp.WriteJSON(w, http.StatusCreated, state)
}

type checkPasswordRequestBody struct {
	Password string `json:"password"`
}

func checkPasswordRequestFromHTTP(r *http.Request) (checkPasswordRequestBody, error) {
	var body checkPasswordRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return checkPasswordRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}
	if body.Password == "" {
		return checkPasswordRequestBody{}, errors.New("password must not be empty")
	}
	return body, nil
}

func (h Handler) checkPassword(w http.ResponseWriter, r *http.Request) {
	body, err := checkPasswordRequestFromHTTP(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
		return
	}
	state, err := h.service.checkPassword(r.Context(), body.Password)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	jsonresp.WriteJSON(w, http.StatusOK, state)
}

type checkTOTPRequestBody struct {
	Code string `json:"code"`
}

func checkTOTPRequestFromHTTP(r *http.Request) (checkTOTPRequestBody, error) {
	var body checkTOTPRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return checkTOTPRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}
	if body.Code == "" {
		return checkTOTPRequestBody{}, errors.New("code must not be empty")
	}
	return body, nil
}

func (h Handler) checkTOTP(w http.ResponseWriter, r *http.Request) {
	body, err := checkTOTPRequestFromHTTP(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
		return
	}
	state, err := h.service.checkTOTP(r.Context(), body.Code)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	jsonresp.WriteJSON(w, http.StatusOK, state)
}

type finalizeRequestBody struct {
	AuthRequestID string `json:"authRequestId"`
}

type finalizeResponseBody struct {
	RedirectURL string `json:"redirectUrl"`
}

func finalizeRequestFromHTTP(r *http.Request) (finalizeRequestBody, error) {
	var body finalizeRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return finalizeRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}
	if body.AuthRequestID == "" {
		return finalizeRequestBody{}, errors.New("authRequestId must not be empty")
	}
	return body, nil
}

func (h Handler) finalize(w http.ResponseWriter, r *http.Request) {
	body, err := finalizeRequestFromHTTP(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
		return
	}
	redirectTo, err := h.service.finalize(r.Context(), body.AuthRequestID)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	jsonresp.WriteJSON(w, http.StatusOK, finalizeResponseBody{RedirectURL: redirectTo})
}

// httpStatusForError maps this slice's error outcomes to HTTP status codes.
func httpStatusForError(err error) int {
	switch {
	case errors.Is(err, errInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, errNoActiveLoginSession):
		return http.StatusConflict
	default:
		return http.StatusBadGateway
	}
}
