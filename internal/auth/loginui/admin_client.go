package loginui

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
)

// adminAPIClient calls ZITADEL's v2 User Service REST API for operations
// that exceed IAM_LOGIN_CLIENT's scope: registering new users, resetting
// passwords, and enrolling TOTP. It authenticates with a separate,
// broader-than-login credential (see AdminTokenSource in New) — the same
// admin-provisioner PAT bff/README.md and scripts/provision-zitadel-bff.sh
// already use for management-API calls.
type adminAPIClient struct{ *restClient }

type registerHumanRequest struct {
	Username string                `json:"username"`
	Profile  registerHumanProfile  `json:"profile"`
	Email    registerHumanEmail    `json:"email"`
	Password registerHumanPassword `json:"password"`
}

type registerHumanProfile struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
}

// returnCode is an empty JSON object marker (ZITADEL's oneof convention: the
// selected alternative's own field name carries its value, `{}` here since
// ReturnCode itself has no fields).
type returnCode struct{}

type registerHumanEmail struct {
	Email      string      `json:"email"`
	ReturnCode *returnCode `json:"returnCode"`
}

type registerHumanPassword struct {
	Password       string `json:"password"`
	ChangeRequired bool   `json:"changeRequired"`
}

type registerHumanResponse struct {
	UserID    string `json:"userId"`
	EmailCode string `json:"emailCode"`
}

// registerHuman creates a new ZITADEL user with a password already set and
// requests email verification in "return code" mode: ZITADEL hands the code
// back in the response instead of emailing it itself, so this app's own
// mailer can send it (see registration.go).
func (c *adminAPIClient) registerHuman(
	ctx context.Context, loginName, email, password, givenName, familyName string,
) (string, string, error) {
	var resp registerHumanResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v2/users/human", registerHumanRequest{
		Username: loginName,
		Profile:  registerHumanProfile{GivenName: givenName, FamilyName: familyName},
		Email:    registerHumanEmail{Email: email, ReturnCode: &returnCode{}},
		Password: registerHumanPassword{Password: password, ChangeRequired: false},
	}, &resp); err != nil {
		return "", "", err
	}
	if resp.UserID == "" || resp.EmailCode == "" {
		return "", "", fmt.Errorf("register human: empty user id or email code")
	}
	return resp.UserID, resp.EmailCode, nil
}

func (c *adminAPIClient) verifyEmail(ctx context.Context, userID, code string) error {
	return c.doJSON(ctx, http.MethodPost,
		"/v2/users/"+url.PathEscape(userID)+"/email/verify",
		map[string]string{"verificationCode": code}, nil,
	)
}

type passwordResetResponse struct {
	VerificationCode string `json:"verificationCode"`
}

// requestPasswordReset asks ZITADEL for a password-reset code in "return
// code" mode, same rationale as registerHuman's email verification.
func (c *adminAPIClient) requestPasswordReset(ctx context.Context, userID string) (string, error) {
	var resp passwordResetResponse
	if err := c.doJSON(ctx, http.MethodPost,
		"/v2/users/"+url.PathEscape(userID)+"/password_reset",
		map[string]*returnCode{"returnCode": {}}, &resp,
	); err != nil {
		return "", err
	}
	if resp.VerificationCode == "" {
		return "", fmt.Errorf("request password reset: empty verification code")
	}
	return resp.VerificationCode, nil
}

func (c *adminAPIClient) setPassword(ctx context.Context, userID, newPassword, verificationCode string) error {
	return c.doJSON(ctx, http.MethodPost, "/v2/users/"+url.PathEscape(userID)+"/password", map[string]any{
		"newPassword":      registerHumanPassword{Password: newPassword, ChangeRequired: false},
		"verificationCode": verificationCode,
	}, nil)
}

type findUsersResponse struct {
	Result []struct {
		UserID string `json:"userId"`
		Human  struct {
			Email struct {
				Email string `json:"email"`
			} `json:"email"`
		} `json:"human"`
	} `json:"result"`
}

// findUserByLoginName resolves a login name to ZITADEL's internal user ID
// and email, needed for the admin-scoped calls above (which take a user ID,
// unlike the Session API's loginName-based checks) and for emailing a
// password-reset link. Returns errInvalidCredentials if no user matches, so
// callers can respond identically to "wrong password" and avoid leaking
// account existence.
func (c *adminAPIClient) findUserByLoginName(ctx context.Context, loginName string) (string, string, error) {
	var resp findUsersResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v2/users", map[string]any{
		"queries": []map[string]any{{"loginNameQuery": map[string]string{"loginName": loginName}}},
	}, &resp); err != nil {
		return "", "", err
	}
	if len(resp.Result) == 0 {
		return "", "", fmt.Errorf("%w: no user found for login name", errInvalidCredentials)
	}
	return resp.Result[0].UserID, resp.Result[0].Human.Email.Email, nil
}

type registerTOTPResponse struct {
	URI    string `json:"uri"`
	Secret string `json:"secret"`
}

// registerTOTP starts TOTP enrollment for userID, returning the otpauth: URI
// (for a QR code, if rendered) and the raw secret (for manual entry).
// Enrollment isn't active until verifyTOTPRegistration succeeds.
func (c *adminAPIClient) registerTOTP(ctx context.Context, userID string) (string, string, error) {
	var resp registerTOTPResponse
	if err := c.doJSON(ctx, http.MethodPost,
		"/v2/users/"+url.PathEscape(userID)+"/totp", map[string]any{}, &resp,
	); err != nil {
		return "", "", err
	}
	return resp.URI, resp.Secret, nil
}

func (c *adminAPIClient) verifyTOTPRegistration(ctx context.Context, userID, code string) error {
	return c.doJSON(ctx, http.MethodPost,
		"/v2/users/"+url.PathEscape(userID)+"/totp/verify",
		map[string]string{"code": code}, nil,
	)
}

type authenticationMethodsResponse struct {
	AuthMethodTypes []string `json:"authMethodTypes"`
}

// hasTOTP reports whether userID has completed TOTP enrollment, so the
// login flow knows to require a checks.totp step.
func (c *adminAPIClient) hasTOTP(ctx context.Context, userID string) (bool, error) {
	var resp authenticationMethodsResponse
	if err := c.doJSON(ctx, http.MethodGet,
		"/v2/users/"+url.PathEscape(userID)+"/authentication_methods", nil, &resp,
	); err != nil {
		return false, err
	}
	return slices.Contains(resp.AuthMethodTypes, "AUTHENTICATION_METHOD_TYPE_TOTP"), nil
}
