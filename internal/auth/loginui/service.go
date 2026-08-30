// Package loginui implements a custom, self-hosted login UI backed by
// ZITADEL's Session API (REST v2), as an alternative to redirecting the
// browser to ZITADEL's hosted login page. See README.md for the full flow.
package loginui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
)

const (
	loginNameKey     = "loginui.login_name"
	zitadelUserIDKey = "loginui.user_id"
	zitadelSessionID = "loginui.session_id"
	zitadelToken     = "loginui.session_token"
	passwordOKKey    = "loginui.password_verified"
	needsTOTPKey     = "loginui.needs_totp"
	totpOKKey        = "loginui.totp_verified"
)

var (
	// errNoActiveLoginSession is returned when a request needs an in-progress
	// login transaction (e.g. to check a password, or to finalize) but none
	// was found, it expired, or it isn't sufficient yet. The caller must
	// start over with POST /auth/session.
	errNoActiveLoginSession = errors.New("no active login session")
	// errInvalidCredentials wraps any check ZITADEL rejected (unknown login
	// name, wrong password, wrong TOTP code), so the HTTP layer can answer
	// with 401 without leaking which factor was wrong.
	errInvalidCredentials = errors.New("invalid credentials")
)

// Service drives ZITADEL's Session API to authenticate users through a
// custom frontend, then finalizes the OIDC authorization request ZITADEL
// redirected the browser to our login UI for. It also owns the
// registration, email-verification, password-reset, and TOTP-enrollment
// flows in registration.go.
type Service struct {
	sessionAPI      *sessionAPIClient
	oidcAPI         *oidcAuthRequestClient
	adminAPI        *adminAPIClient
	bff             AuthorizationCompleter
	mailer          Mailer
	frontendBaseURL string
	transactions    *scs.SessionManager
}

// TokenSource returns a valid bearer token for calling ZITADEL's Session and
// OIDC v2 REST APIs. It is an interface so tests can supply a fixed token
// without a real credential.
type TokenSource interface {
	GetValidToken() (string, error)
}

// StaticToken is a TokenSource for a pre-issued, long-lived personal access
// token (PAT) read once at startup — e.g. the IAM_LOGIN_CLIENT-scoped PAT
// ZITADEL's `FirstInstance.Org.LoginClient` bootstrap writes to the shared
// bootstrap volume in local Compose (see README.md). It is not suitable for
// a token that needs refreshing.
type StaticToken string

// GetValidToken implements TokenSource.
func (t StaticToken) GetValidToken() (string, error) { return string(t), nil }

// AuthorizationCompleter completes an OIDC authorization-code callback and
// establishes the browser's app session. *bff.Service satisfies this; it is
// an interface here so tests can substitute a stub instead of standing up a
// full BFF service (OIDC discovery, database, HTTP transport).
type AuthorizationCompleter interface {
	CompleteAuthorizationCallback(ctx context.Context, code, state string) (string, error)
}

// New wires the Session-API login UI.
//
//   - httpClient/issuer reach ZITADEL's v2 REST API; pass bffService.HTTPClient()
//     and bffService.Issuer() to reuse the BFF's private-address-aware dialer.
//   - loginTokens supplies the bearer credential for Session/OIDC-auth-request
//     calls; it must belong to a ZITADEL account granted the IAM_LOGIN_CLIENT
//     role. In local Compose this is StaticToken wrapping the PAT ZITADEL's
//     bootstrap writes for exactly this purpose. See README.md.
//   - adminTokens supplies the bearer credential for registration/password-reset/
//     TOTP-enrollment calls, which need broader permissions than
//     IAM_LOGIN_CLIENT. In local Compose this is the same admin-provisioner
//     PAT bff/README.md documents.
//   - mailer sends verification/reset emails this slice owns delivering
//     itself (see registration.go); frontendBaseURL is the origin those
//     emails' links point back to.
//   - sessions is the app's shared, PostgreSQL-backed SCS store (reused the
//     same way bff.New reuses it for its own short-lived OAuth transaction).
//   - bffService supplies the shared authorization-code completion logic
//     once a ZITADEL session is sufficient to finalize.
func New(
	cookieSecure bool,
	httpClient *http.Client,
	issuer string,
	loginTokens TokenSource,
	adminTokens TokenSource,
	mailer Mailer,
	frontendBaseURL string,
	sessions *scs.SessionManager,
	bffService AuthorizationCompleter,
) *Service {
	transactions := scs.New()
	transactions.Store = sessions.Store
	transactions.HashTokenInStore = true
	transactions.Lifetime = 10 * time.Minute
	transactions.Cookie.Name = "login_session"
	transactions.Cookie.HttpOnly = true
	transactions.Cookie.SameSite = http.SameSiteLaxMode
	transactions.Cookie.Secure = cookieSecure
	transactions.Cookie.Path = "/auth/session"
	loginRest := &restClient{http: httpClient, issuer: issuer, tokens: loginTokens}
	adminRest := &restClient{http: httpClient, issuer: issuer, tokens: adminTokens}
	return &Service{
		sessionAPI:      &sessionAPIClient{loginRest},
		oidcAPI:         &oidcAuthRequestClient{loginRest},
		adminAPI:        &adminAPIClient{adminRest},
		bff:             bffService,
		mailer:          mailer,
		frontendBaseURL: frontendBaseURL,
		transactions:    transactions,
	}
}

// TransactionMiddleware loads the login transaction cookie that tracks the
// in-progress ZITADEL session across the multi-step login form.
func (s *Service) TransactionMiddleware(next http.Handler) http.Handler {
	return s.transactions.LoadAndSave(next)
}

// startWithLoginName creates a new ZITADEL session for loginName and stores
// its id/token server-side, along with whether the account requires a TOTP
// check. The returned State never reports Sufficient immediately after
// creation, since only the user factor is verified.
func (s *Service) startWithLoginName(ctx context.Context, loginName string) (State, error) {
	sessionID, token, err := s.sessionAPI.createSession(ctx, loginName)
	if err != nil {
		return State{}, err
	}
	userID, _, err := s.adminAPI.findUserByLoginName(ctx, loginName)
	if err != nil {
		return State{}, err
	}
	needsTOTP, err := s.adminAPI.hasTOTP(ctx, userID)
	if err != nil {
		return State{}, fmt.Errorf("check totp enrollment: %w", err)
	}
	if err := s.transactions.RenewToken(ctx); err != nil {
		return State{}, fmt.Errorf("renew login transaction: %w", err)
	}
	s.transactions.Put(ctx, loginNameKey, loginName)
	s.transactions.Put(ctx, zitadelUserIDKey, userID)
	s.transactions.Put(ctx, zitadelSessionID, sessionID)
	s.transactions.Put(ctx, zitadelToken, token)
	s.transactions.Put(ctx, needsTOTPKey, needsTOTP)
	return s.state(ctx), nil
}

// checkPassword verifies password against the session started by
// startWithLoginName. It requires that session to still be active.
func (s *Service) checkPassword(ctx context.Context, password string) (State, error) {
	sessionID := s.transactions.GetString(ctx, zitadelSessionID)
	if sessionID == "" {
		return State{}, errNoActiveLoginSession
	}
	// ZITADEL invalidates the previous session token on every successful
	// check and issues a new one; the old token can no longer be used.
	newToken, err := s.sessionAPI.checkPassword(ctx, sessionID, password)
	if err != nil {
		return State{}, err
	}
	s.transactions.Put(ctx, zitadelToken, newToken)
	s.transactions.Put(ctx, passwordOKKey, true)
	return s.state(ctx), nil
}

// checkTOTP verifies a TOTP code against the session started by
// startWithLoginName. It requires a password check to have already
// succeeded, matching ZITADEL's own requirement that the user factor (and
// typically password) precede a second factor.
func (s *Service) checkTOTP(ctx context.Context, code string) (State, error) {
	sessionID := s.transactions.GetString(ctx, zitadelSessionID)
	if sessionID == "" || !s.transactions.GetBool(ctx, passwordOKKey) {
		return State{}, errNoActiveLoginSession
	}
	newToken, err := s.sessionAPI.checkTOTP(ctx, sessionID, code)
	if err != nil {
		return State{}, err
	}
	s.transactions.Put(ctx, zitadelToken, newToken)
	s.transactions.Put(ctx, totpOKKey, true)
	return s.state(ctx), nil
}

func (s *Service) state(ctx context.Context) State {
	return State{
		LoginName:        s.transactions.GetString(ctx, loginNameKey),
		PasswordVerified: s.transactions.GetBool(ctx, passwordOKKey),
		NeedsTOTP:        s.transactions.GetBool(ctx, needsTOTPKey),
		TotpVerified:     s.transactions.GetBool(ctx, totpOKKey),
	}
}

// finalize links the ZITADEL session accumulated so far to authRequestID and,
// once ZITADEL accepts it, completes the same authorization-code exchange
// the classic redirect-based callback performs.
func (s *Service) finalize(ctx context.Context, authRequestID string) (string, error) {
	if !s.state(ctx).Sufficient() {
		return "", errNoActiveLoginSession
	}
	sessionID := s.transactions.GetString(ctx, zitadelSessionID)
	sessionToken := s.transactions.GetString(ctx, zitadelToken)
	if err := s.transactions.Destroy(ctx); err != nil {
		return "", fmt.Errorf("destroy login transaction: %w", err)
	}
	code, callbackState, err := s.oidcAPI.createCallback(ctx, authRequestID, sessionID, sessionToken)
	if err != nil {
		return "", fmt.Errorf("finalize auth request: %w", err)
	}
	redirectTo, err := s.bff.CompleteAuthorizationCallback(ctx, code, callbackState)
	if err != nil {
		return "", fmt.Errorf("complete authorization callback: %w", err)
	}
	return redirectTo, nil
}
