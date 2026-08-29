package bff

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/mhmdkzr/app/internal/config"
)

// tokenEndpointStub records the last request form body it received and
// replies with a fixed access token response.
func tokenEndpointStub(t *testing.T) (*httptest.Server, *url.Values) {
	t.Helper()
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse token request form: %v", err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(oidc.AccessTokenResponse{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	t.Cleanup(server.Close)
	return server, &gotForm
}

func TestExchangeSendsAuthorizationCodeGrantType(t *testing.T) {
	t.Parallel()
	server, gotForm := tokenEndpointStub(t)
	service := &Service{
		cfg: config.AuthConfig{
			ClientID:     "client",
			ClientSecret: "secret",
			RedirectURL:  "https://app.example/auth/callback",
		},
		discovery: &oidc.DiscoveryConfiguration{TokenEndpoint: server.URL},
		http:      server.Client(),
	}
	accessToken, refreshToken, _, err := service.exchange(t.Context(), "auth-code", "verifier")
	if err != nil {
		t.Fatal(err)
	}
	if accessToken != "access-token" || refreshToken != "refresh-token" {
		t.Fatalf("exchange() = (%q, %q), want (access-token, refresh-token)", accessToken, refreshToken)
	}
	if got := gotForm.Get("grant_type"); got != string(oidc.GrantTypeCode) {
		t.Fatalf("grant_type = %q, want %q", got, oidc.GrantTypeCode)
	}
	if got := gotForm.Get("code"); got != "auth-code" {
		t.Fatalf("code = %q, want auth-code", got)
	}
	if got := gotForm.Get("code_verifier"); got != "verifier" {
		t.Fatalf("code_verifier = %q, want verifier", got)
	}
}

func TestRefreshSendsRefreshTokenGrantType(t *testing.T) {
	t.Parallel()
	server, gotForm := tokenEndpointStub(t)
	sessions := scs.New()
	service := &Service{
		cfg:       config.AuthConfig{ClientID: "client", ClientSecret: "secret"},
		discovery: &oidc.DiscoveryConfiguration{TokenEndpoint: server.URL},
		http:      server.Client(),
		sessions:  sessions,
	}
	ctx, err := sessions.Load(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	accessToken, err := service.refresh(ctx, "old-refresh-token")
	if err != nil {
		t.Fatal(err)
	}
	if accessToken != "access-token" {
		t.Fatalf("refresh() = %q, want access-token", accessToken)
	}
	if got := gotForm.Get("grant_type"); got != string(oidc.GrantTypeRefreshToken) {
		t.Fatalf("grant_type = %q, want %q", got, oidc.GrantTypeRefreshToken)
	}
	if got := gotForm.Get("refresh_token"); got != "old-refresh-token" {
		t.Fatalf("refresh_token = %q, want old-refresh-token", got)
	}
}

func TestSafeReturnTo(t *testing.T) {
	t.Parallel()
	for input, want := range map[string]string{"": "/", "//evil.example": "/", "https://evil.example": "/", "/dashboard": "/dashboard"} {
		if got := safeReturnTo(input); got != want {
			t.Errorf("safeReturnTo(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestOIDCHTTPClientDialsPrivateAddressAndPreservesIssuerHost(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "issuer.example:8080" {
			t.Errorf("Host = %q, want public issuer host", r.Host)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client, err := oidcHTTPClient(config.AuthConfig{
		Issuer:          "http://issuer.example:8080",
		InternalAddress: server.Listener.Addr().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Get("http://issuer.example:8080/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func TestLoginUsesStrictShortLivedSessionAndPKCE(t *testing.T) {
	t.Parallel()
	sessions := scs.New()
	service := &Service{
		cfg:          config.AuthConfig{ClientID: "client", RedirectURL: "https://app.example/auth/callback"},
		sessions:     sessions,
		transactions: sessions,
		discovery:    &oidc.DiscoveryConfiguration{AuthorizationEndpoint: "https://id.example/authorize"},
	}
	handler := NewHandler(service)
	request := httptest.NewRequest(http.MethodGet, "/auth/login?return_to=//evil.example", nil)
	ctx, err := sessions.Load(request.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.login(recorder, request.WithContext(ctx))
	if recorder.Code != 302 {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	redirect, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if redirect.Query().Get("code_challenge_method") != "S256" || redirect.Query().Get("code_challenge") == "" {
		t.Fatal("missing S256 PKCE challenge")
	}
	if sessions.GetString(ctx, returnToKey) != "/" {
		t.Fatalf("return destination = %q, want /", sessions.GetString(ctx, returnToKey))
	}
	if sessions.GetString(ctx, stateKey) == "" || sessions.GetString(ctx, verifierKey) == "" ||
		sessions.GetString(ctx, nonceKey) == "" {
		t.Fatal("missing callback state")
	}
}

func TestCallbackRejectsMissingStoredState(t *testing.T) {
	t.Parallel()
	sessions := scs.New()
	handler := NewHandler(&Service{sessions: sessions, transactions: sessions})
	request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=code&state=state", nil)
	ctx, err := sessions.Load(request.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.callback(recorder, request.WithContext(ctx))
	if recorder.Code != 400 {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestCallbackRejectsMismatchedStoredState(t *testing.T) {
	t.Parallel()
	sessions := scs.New()
	handler := NewHandler(&Service{sessions: sessions, transactions: sessions})
	request := httptest.NewRequest(http.MethodGet, "/auth/callback?code=code&state=wrong", nil)
	ctx, err := sessions.Load(request.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	sessions.Put(ctx, stateKey, "expected")
	recorder := httptest.NewRecorder()
	handler.callback(recorder, request.WithContext(ctx))
	if recorder.Code != 400 {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestLogoutRejectsRequestWithoutCSRFHeader(t *testing.T) {
	t.Parallel()
	sessions := scs.New()
	handler := NewHandler(&Service{sessions: sessions})
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	ctx, err := sessions.Load(request.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.logout(recorder, request.WithContext(ctx))
	if recorder.Code != 403 {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestLogoutXHRDestroysLocalSessionWithoutFollowingEndSession(t *testing.T) {
	t.Parallel()
	sessions := scs.New()
	handler := NewHandler(&Service{sessions: sessions})
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	ctx, err := sessions.Load(request.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	sessions.Put(ctx, userIDKey, "user")
	recorder := httptest.NewRecorder()
	handler.logout(recorder, request.WithContext(ctx))
	if recorder.Code != 204 {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if sessions.Exists(ctx, userIDKey) {
		t.Fatal("session data remains after logout")
	}
}

func TestLogoutXHRReturnsEndSessionNavigation(t *testing.T) {
	t.Parallel()
	sessions := scs.New()
	handler := NewHandler(&Service{
		cfg:       config.AuthConfig{ClientID: "client", PostLogoutRedirectURL: "https://app.example/"},
		sessions:  sessions,
		discovery: &oidc.DiscoveryConfiguration{EndSessionEndpoint: "https://id.example/logout"},
	})
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	ctx, err := sessions.Load(request.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.logout(recorder, request.WithContext(ctx))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	redirect, err := url.Parse(response.RedirectURL)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if redirect.String() != "https://id.example/logout?client_id=client&post_logout_redirect_uri=https%3A%2F%2Fapp.example%2F" {
		t.Fatalf("redirect URL = %q", redirect)
	}
}
