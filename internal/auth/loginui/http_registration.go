package loginui

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mhmdkzr/app/pkg/jsonresp"
)

// RegistrationHandler exposes the browser-facing endpoints the custom
// Svelte registration/password-reset/TOTP-enrollment forms call. It is
// separate from Handler (session-based login) since these endpoints don't
// need an in-progress login transaction — registration and password reset
// happen before any ZITADEL session exists.
type RegistrationHandler struct{ service *Service }

// NewRegistrationHandler creates a registration/password-reset handler.
func NewRegistrationHandler(service *Service) RegistrationHandler {
	return RegistrationHandler{service: service}
}

// Register creates a new account and emails a verification link.
func (h RegistrationHandler) Register(w http.ResponseWriter, r *http.Request) { h.register(w, r) }

// VerifyEmail confirms a registration's emailed verification code.
func (h RegistrationHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) { h.verifyEmail(w, r) }

// RequestPasswordReset emails a password-reset link.
func (h RegistrationHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	h.requestPasswordReset(w, r)
}

// ResetPassword sets a new password using an emailed reset code.
func (h RegistrationHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	h.resetPassword(w, r)
}

// StartTOTPEnrollment begins TOTP enrollment for a just-registered account.
func (h RegistrationHandler) StartTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	h.startTOTPEnrollment(w, r)
}

// ConfirmTOTPEnrollment completes TOTP enrollment.
func (h RegistrationHandler) ConfirmTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	h.confirmTOTPEnrollment(w, r)
}

type registerRequestBody struct {
	LoginName  string `json:"loginName"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

type registerResponseBody struct {
	UserID string `json:"userId"`
}

func registerRequestFromHTTP(r *http.Request) (registerRequestBody, error) {
	var body registerRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return registerRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}
	for name, value := range map[string]string{
		"loginName": body.LoginName, "email": body.Email, "password": body.Password,
		"givenName": body.GivenName, "familyName": body.FamilyName,
	} {
		if value == "" {
			return registerRequestBody{}, fmt.Errorf("%s must not be empty", name)
		}
	}
	return body, nil
}

func (h RegistrationHandler) register(w http.ResponseWriter, r *http.Request) {
	body, err := registerRequestFromHTTP(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
		return
	}
	userID, err := h.service.Register(
		r.Context(), body.LoginName, body.Email, body.Password, body.GivenName, body.FamilyName,
	)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	jsonresp.WriteJSON(w, http.StatusCreated, registerResponseBody{UserID: userID})
}

type verifyEmailRequestBody struct {
	UserID string `json:"userId"`
	Code   string `json:"code"`
}

func verifyEmailRequestFromHTTP(r *http.Request) (verifyEmailRequestBody, error) {
	var body verifyEmailRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return verifyEmailRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}
	if body.UserID == "" || body.Code == "" {
		return verifyEmailRequestBody{}, errors.New("userId and code must not be empty")
	}
	return body, nil
}

func (h RegistrationHandler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	body, err := verifyEmailRequestFromHTTP(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.service.VerifyEmail(r.Context(), body.UserID, body.Code); err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type requestPasswordResetBody struct {
	LoginName string `json:"loginName"`
}

func requestPasswordResetFromHTTP(r *http.Request) (requestPasswordResetBody, error) {
	var body requestPasswordResetBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return requestPasswordResetBody{}, fmt.Errorf("decode request body: %w", err)
	}
	if body.LoginName == "" {
		return requestPasswordResetBody{}, errors.New("loginName must not be empty")
	}
	return body, nil
}

func (h RegistrationHandler) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	body, err := requestPasswordResetFromHTTP(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.service.RequestPasswordReset(r.Context(), body.LoginName); err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type resetPasswordRequestBody struct {
	UserID      string `json:"userId"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

func resetPasswordRequestFromHTTP(r *http.Request) (resetPasswordRequestBody, error) {
	var body resetPasswordRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return resetPasswordRequestBody{}, fmt.Errorf("decode request body: %w", err)
	}
	for name, value := range map[string]string{"userId": body.UserID, "code": body.Code, "newPassword": body.NewPassword} {
		if value == "" {
			return resetPasswordRequestBody{}, fmt.Errorf("%s must not be empty", name)
		}
	}
	return body, nil
}

func (h RegistrationHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	body, err := resetPasswordRequestFromHTTP(r)
	if err != nil {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.service.ResetPassword(r.Context(), body.UserID, body.Code, body.NewPassword); err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type startTOTPEnrollmentRequestBody struct {
	UserID string `json:"userId"`
}

type startTOTPEnrollmentResponseBody struct {
	URI    string `json:"uri"`
	Secret string `json:"secret"`
}

func (h RegistrationHandler) startTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	var body startTOTPEnrollmentRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, errors.New("userId must not be empty"))
		return
	}
	uri, secret, err := h.service.StartTOTPEnrollment(r.Context(), body.UserID)
	if err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	jsonresp.WriteJSON(w, http.StatusCreated, startTOTPEnrollmentResponseBody{URI: uri, Secret: secret})
}

type confirmTOTPEnrollmentRequestBody struct {
	UserID string `json:"userId"`
	Code   string `json:"code"`
}

func (h RegistrationHandler) confirmTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	var body confirmTOTPEnrollmentRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" || body.Code == "" {
		jsonresp.WriteHTTPError(w, http.StatusBadRequest, errors.New("userId and code must not be empty"))
		return
	}
	if err := h.service.ConfirmTOTPEnrollment(r.Context(), body.UserID, body.Code); err != nil {
		jsonresp.WriteHTTPError(w, httpStatusForError(err), err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
