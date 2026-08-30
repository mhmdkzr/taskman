package bff

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/client"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/mhmdkzr/app/internal/identity"
	"github.com/mhmdkzr/app/internal/identity/provision"
)

var (
	errUnauthenticated = errors.New("unauthenticated")
	errInvalidCallback = errors.New("invalid authentication callback")
)

type principalContextKey struct{}

// Principal returns the authenticated app-owned user identity.
func Principal(ctx context.Context) (identity.UserID, bool) {
	id, ok := ctx.Value(principalContextKey{}).(identity.UserID)
	return id, ok
}

// CompleteAuthorizationCallback validates an OIDC authorization-code callback
// against the stored transaction (state, PKCE verifier), exchanges the code,
// provisions/loads the app-owned identity, and establishes the browser's app
// session. It returns the safe post-login redirect target.
//
// It is shared by the classic top-level redirect callback (bff/http.go) and
// the custom Session-API login UI (auth/loginui), which reaches the same
// point via ZITADEL's CreateCallback rather than a browser redirect.
func (s *Service) CompleteAuthorizationCallback(ctx context.Context, code, state string) (string, error) {
	expectedState := s.transactions.GetString(ctx, stateKey)
	if code == "" || expectedState == "" ||
		subtle.ConstantTimeCompare([]byte(state), []byte(expectedState)) != 1 {
		return "", errInvalidCallback
	}
	verifier := s.transactions.GetString(ctx, verifierKey)
	returnTo := safeReturnTo(s.transactions.GetString(ctx, returnToKey))
	if err := s.transactions.Destroy(ctx); err != nil {
		return "", fmt.Errorf("destroy auth transaction: %w", err)
	}
	accessToken, refreshToken, expiry, err := s.exchange(ctx, code, verifier)
	if err != nil {
		return "", err
	}
	subject, err := s.introspect(ctx, accessToken)
	if err != nil {
		return "", err
	}
	userID, err := provision.FindOrCreate(ctx, s.db, subject)
	if err != nil {
		return "", fmt.Errorf("provision app identity: %w", err)
	}
	if err := s.sessions.RenewToken(ctx); err != nil {
		return "", fmt.Errorf("renew session: %w", err)
	}
	s.sessions.Put(ctx, accessTokenKey, accessToken)
	s.sessions.Put(ctx, refreshTokenKey, refreshToken)
	s.sessions.Put(ctx, expiryKey, expiry)
	s.sessions.Put(ctx, userIDKey, userID.String())
	return returnTo, nil
}

func (s *Service) introspect(ctx context.Context, token string) (identity.ZitadelSubject, error) {
	form := url.Values{"token": {token}}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.discovery.IntrospectionEndpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("create introspection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.cfg.ClientID, s.cfg.ClientSecret)
	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("introspect access token: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("close introspection response body", "error", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("introspection returned %s", resp.Status)
	}
	var result struct {
		Active  bool   `json:"active"`
		Subject string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode introspection: %w", err)
	}
	if !result.Active {
		return "", errUnauthenticated
	}
	subject, err := identity.NewZitadelSubject(result.Subject)
	if err != nil {
		return "", fmt.Errorf("parse introspected subject: %w", err)
	}
	return subject, nil
}

func (s *Service) principal(r *http.Request) (identity.UserID, error) {
	if !s.sessions.Exists(r.Context(), userIDKey) {
		return identity.UserID{}, errUnauthenticated
	}
	expiry := s.sessions.GetTime(r.Context(), expiryKey)
	accessToken := s.sessions.GetString(r.Context(), accessTokenKey)
	if expiry.Before(time.Now().Add(s.cfg.RefreshLeeway)) {
		var err error
		accessToken, err = s.refresh(r.Context(), s.sessions.GetString(r.Context(), refreshTokenKey))
		if err != nil {
			return identity.UserID{}, errUnauthenticated
		}
	}
	if _, err := s.introspect(r.Context(), accessToken); err != nil {
		return identity.UserID{}, errUnauthenticated
	}
	id, err := uuid.Parse(s.sessions.GetString(r.Context(), userIDKey))
	if err != nil {
		return identity.UserID{}, errUnauthenticated
	}
	return identity.UserID(id), nil
}

// refreshTokenRequest mirrors oidc.RefreshTokenRequest but adds an explicit
// grant_type field: per that type's own doc comment, it "is not useful for
// making refresh requests because the grant_type is not included explicitly".
type refreshTokenRequest struct {
	GrantType    oidc.GrantType `schema:"grant_type"`
	RefreshToken string         `schema:"refresh_token"`
	ClientID     string         `schema:"client_id"`
}

func (s *Service) refresh(ctx context.Context, refreshToken string) (string, error) {
	if refreshToken == "" {
		return "", errUnauthenticated
	}
	token, err := client.CallTokenEndpointWithAuthFn(ctx, &refreshTokenRequest{
		GrantType:    oidc.GrantTypeRefreshToken,
		RefreshToken: refreshToken,
		ClientID:     s.cfg.ClientID,
	}, httphelper.AuthorizeBasic(s.cfg.ClientID, s.cfg.ClientSecret),
		endpointCaller{
			endpoint: s.discovery.TokenEndpoint,
			http:     s.http,
		})
	if err != nil {
		return "", fmt.Errorf("refresh access token: %w", err)
	}
	if token.AccessToken == "" {
		return "", errUnauthenticated
	}
	s.sessions.Put(ctx, accessTokenKey, token.AccessToken)
	if token.RefreshToken != "" {
		s.sessions.Put(ctx, refreshTokenKey, token.RefreshToken)
	}
	s.sessions.Put(ctx, expiryKey, token.Expiry)
	return token.AccessToken, nil
}
