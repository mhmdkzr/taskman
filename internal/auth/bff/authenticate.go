package bff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mhmdkzr/app/internal/identity"
	"github.com/zitadel/oidc/v3/pkg/client"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

var errUnauthenticated = errors.New("unauthenticated")

type principalContextKey struct{}

// Principal returns the authenticated app-owned user identity.
func Principal(ctx context.Context) (identity.UserID, bool) {
	id, ok := ctx.Value(principalContextKey{}).(identity.UserID)
	return id, ok
}

func (s *Service) introspect(ctx context.Context, token string) (identity.ZitadelSubject, error) {
	form := url.Values{"token": {token}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.discovery.IntrospectionEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create introspection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.cfg.ClientID, s.cfg.ClientSecret)
	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("introspect access token: %w", err)
	}
	defer resp.Body.Close()
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
	return identity.NewZitadelSubject(result.Subject)
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

func (s *Service) refresh(ctx context.Context, refreshToken string) (string, error) {
	if refreshToken == "" {
		return "", errUnauthenticated
	}
	token, err := client.CallTokenEndpointWithAuthFn(ctx, &oidc.RefreshTokenRequest{
		RefreshToken: refreshToken, ClientID: s.cfg.ClientID,
	}, httphelper.AuthorizeBasic(s.cfg.ClientID, s.cfg.ClientSecret), endpointCaller{endpoint: s.discovery.TokenEndpoint, http: s.http})
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
