// Package bff implements the backend-for-frontend authentication boundary.
package bff

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/zitadel/oidc/v3/pkg/client"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/mhmdkzr/app/internal/config"
)

const (
	stateKey        = "auth.state"
	verifierKey     = "auth.verifier"
	nonceKey        = "auth.nonce"
	returnToKey     = "auth.return_to"
	accessTokenKey  = "auth.access_token"
	refreshTokenKey = "auth.refresh_token"
	expiryKey       = "auth.expiry"
	userIDKey       = "auth.user_id"
)

// Service owns the OIDC discovery state and secure app session operations.
type Service struct {
	cfg          config.AuthConfig
	db           *sql.DB
	sessions     *scs.SessionManager
	transactions *scs.SessionManager
	discovery    *oidc.DiscoveryConfiguration
	http         *http.Client
}

// New discovers the configured OIDC provider once at startup.
func New(ctx context.Context, cfg config.AuthConfig, db *sql.DB, sessions *scs.SessionManager) (*Service, error) {
	httpClient, err := oidcHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	discovery, err := client.Discover(ctx, cfg.Issuer, httpClient)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	transactions := scs.New()
	transactions.Store = sessions.Store
	transactions.HashTokenInStore = true
	transactions.Lifetime = 10 * time.Minute
	transactions.Cookie.Name = "auth_transaction"
	transactions.Cookie.HttpOnly = true
	transactions.Cookie.SameSite = http.SameSiteLaxMode
	transactions.Cookie.Secure = cfg.CookieSecure
	transactions.Cookie.Path = "/auth"
	return &Service{cfg: cfg, db: db, sessions: sessions, transactions: transactions, discovery: discovery, http: httpClient}, nil
}

// oidcHTTPClient dials the optional private provider address while retaining
// the public issuer URL (and therefore its Host header) on every request.
// This lets a Compose service reach a locally published identity provider
// without changing the issuer embedded in browser-facing OIDC URLs.
func oidcHTTPClient(cfg config.AuthConfig) (*http.Client, error) {
	if cfg.InternalAddress == "" {
		return http.DefaultClient, nil
	}
	issuer, err := url.Parse(cfg.Issuer)
	if err != nil || issuer.Host == "" {
		return nil, fmt.Errorf("parse OIDC issuer for internal address: %w", err)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address == issuer.Host {
			address = cfg.InternalAddress
		}
		return dialer.DialContext(ctx, network, address)
	}
	return &http.Client{Transport: transport}, nil
}

// TransactionMiddleware loads the short-lived OAuth transaction cookie. It is
// intentionally Lax so the authorization server's top-level callback can return it.
func (s *Service) TransactionMiddleware(next http.Handler) http.Handler {
	return s.transactions.LoadAndSave(next)
}

func (s *Service) authorizationURL(state, verifier, nonce string) string {
	values := url.Values{
		"client_id":             {s.cfg.ClientID},
		"redirect_uri":          {s.cfg.RedirectURL},
		"response_type":         {"code"},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {oidc.NewSHACodeChallenge(verifier)},
		"code_challenge_method": {"S256"},
	}
	return s.discovery.AuthorizationEndpoint + "?" + values.Encode()
}

func randomValue() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read secure random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func safeReturnTo(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}
	return value
}

type endpointCaller struct {
	endpoint string
	http     *http.Client
}

func (e endpointCaller) TokenEndpoint() string    { return e.endpoint }
func (e endpointCaller) HttpClient() *http.Client { return e.http }

func (s *Service) exchange(ctx context.Context, code, verifier string) (accessToken, refreshToken string, expiry time.Time, err error) {
	token, err := client.CallTokenEndpointWithAuthFn(ctx, &oidc.AccessTokenRequest{
		Code: code, RedirectURI: s.cfg.RedirectURL, ClientID: s.cfg.ClientID, CodeVerifier: verifier,
	}, httphelper.AuthorizeBasic(s.cfg.ClientID, s.cfg.ClientSecret), endpointCaller{endpoint: s.discovery.TokenEndpoint, http: s.http})
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("exchange authorization code: %w", err)
	}
	if token.AccessToken == "" {
		return "", "", time.Time{}, fmt.Errorf("authorization code exchange returned no access token")
	}
	return token.AccessToken, token.RefreshToken, token.Expiry, nil
}
