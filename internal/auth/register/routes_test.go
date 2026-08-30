package register

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"

	"github.com/mhmdkzr/app/internal/auth/bff"
	"github.com/mhmdkzr/app/internal/auth/loginui"
	"github.com/mhmdkzr/app/internal/config"
)

// discoveryStub serves a minimal OIDC discovery document so bff.New can
// complete without a live ZITADEL instance.
func discoveryStub(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/oauth/v2/authorize",
			"token_endpoint":         server.URL + "/oauth/v2/token",
			"jwks_uri":               server.URL + "/oauth/v2/keys",
		})
	})
	return server
}

type stubTokens struct{}

func (stubTokens) GetValidToken() (string, error) { return "token", nil }

type stubMailer struct{}

func (stubMailer) Send(context.Context, string, string, string) error { return nil }

// TestFinalizeHandlerChainLoadsBothTransactionContexts is a regression test
// for a panic found via live E2E testing: /auth/session/finalize's handler
// chain must load bff's own auth_transaction-backed context (via
// service.TransactionMiddleware), not just loginUI's login_session one,
// because loginui.Service.finalize calls into
// bff.Service.CompleteAuthorizationCallback, which reads that context. Before
// the fix, this panicked with "scs: no session data in context" instead of
// returning an HTTP error.
func TestFinalizeHandlerChainLoadsBothTransactionContexts(t *testing.T) {
	t.Parallel()
	discovery := discoveryStub(t)
	sessions := scs.New()
	authService, err := bff.New(t.Context(), config.AuthConfig{
		Issuer:      discovery.URL,
		ClientID:    "client",
		RedirectURL: discovery.URL + "/auth/callback",
	}, nil, sessions)
	if err != nil {
		t.Fatalf("bff.New: %v", err)
	}
	loginUI := loginui.New(
		false, discovery.Client(), discovery.URL,
		stubTokens{}, stubTokens{}, stubMailer{}, discovery.URL,
		sessions, authService,
	)
	lh := loginui.NewHandler(loginUI)

	// Mirrors the wrapWithBFFTransaction composition in routes.go exactly.
	handler := authService.TransactionMiddleware(loginUI.TransactionMiddleware(http.HandlerFunc(lh.Finalize)))

	request := httptest.NewRequest(http.MethodPost, "/auth/session/finalize", nil)
	recorder := httptest.NewRecorder()

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("finalize handler chain panicked: %v", r)
			}
		}()
		handler.ServeHTTP(recorder, request)
	}()

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (missing request body), body=%s", recorder.Code, recorder.Body)
	}
}
