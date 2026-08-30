package loginui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
)

const authMethodsWithTOTPResponse = `{"authMethodTypes":["AUTHENTICATION_METHOD_TYPE_PASSWORD","AUTHENTICATION_METHOD_TYPE_TOTP"]}`

type fixedToken string

func (t fixedToken) GetValidToken() (string, error) { return string(t), nil }

// completerStub records the code/state it was called with and returns a
// fixed redirect target, standing in for bff.Service.
type completerStub struct {
	gotCode, gotState string
	redirectTo        string
	err               error
}

func (c *completerStub) CompleteAuthorizationCallback(_ context.Context, code, state string) (string, error) {
	c.gotCode, c.gotState = code, state
	if c.err != nil {
		return "", c.err
	}
	return c.redirectTo, nil
}

// newTestService builds a Service against a stub ZITADEL REST server backed
// by an in-memory session store, mirroring how bff's own tests construct a
// bare Service literal for unit testing. mux's own routes take precedence;
// unhandled requests to the admin-API endpoints startWithLoginName always
// calls (resolve user ID, check TOTP enrollment) fall back to a default
// "found, password-only" response, so tests that don't care about those
// calls don't need to stub them.
func newTestService(
	t *testing.T, mux *http.ServeMux, completer AuthorizationCompleter,
) (*Service, *scs.SessionManager) {
	t.Helper()
	defaults := http.NewServeMux()
	defaults.HandleFunc("POST /v2/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[{"userId":"test-user-id","human":{"email":{"email":"test@example.com"}}}]}`))
	})
	defaults.HandleFunc("GET /v2/users/{id}/authentication_methods", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"authMethodTypes":["AUTHENTICATION_METHOD_TYPE_PASSWORD"]}`))
	})
	combined := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		defaults.ServeHTTP(w, r)
	})
	server := httptest.NewServer(combined)
	t.Cleanup(server.Close)
	rest := &restClient{http: server.Client(), issuer: server.URL, tokens: fixedToken("token")}
	transactions := scs.New()
	return &Service{
		sessionAPI:   &sessionAPIClient{rest},
		oidcAPI:      &oidcAuthRequestClient{rest},
		adminAPI:     &adminAPIClient{rest},
		bff:          completer,
		transactions: transactions,
	}, transactions
}

func loadCtx(t *testing.T, sessions *scs.SessionManager) context.Context {
	t.Helper()
	ctx, err := sessions.Load(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestStartWithLoginNameStoresSessionAndReturnsInsufficientState(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		var body createSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Checks.User.LoginName != "minnie@example.com" {
			t.Fatalf("loginName = %q", body.Checks.User.LoginName)
		}
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	service, sessions := newTestService(t, mux, nil)
	ctx := loadCtx(t, sessions)

	state, err := service.startWithLoginName(ctx, "minnie@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if state.LoginName != "minnie@example.com" || state.PasswordVerified || state.Sufficient() {
		t.Fatalf("state = %+v, want unverified password and not sufficient", state)
	}
	if got := sessions.GetString(ctx, zitadelSessionID); got != "sess-1" {
		t.Fatalf("stored session id = %q", got)
	}
	if got := sessions.GetString(ctx, zitadelToken); got != "tok-1" {
		t.Fatalf("stored session token = %q", got)
	}
}

func TestStartWithLoginNameRejectsUnknownUser(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	service, sessions := newTestService(t, mux, nil)
	ctx := loadCtx(t, sessions)

	if _, err := service.startWithLoginName(ctx, "nobody@example.com"); err == nil {
		t.Fatal("want error for unknown login name")
	}
}

func TestCheckPasswordWithoutActiveSessionFails(t *testing.T) {
	t.Parallel()
	service, sessions := newTestService(t, http.NewServeMux(), nil)
	ctx := loadCtx(t, sessions)

	if _, err := service.checkPassword(ctx, "hunter2"); err == nil {
		t.Fatal("want errNoActiveLoginSession")
	}
}

func TestCheckPasswordSucceedsAfterCreateSession(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	mux.HandleFunc("PATCH /v2/sessions/sess-1", func(w http.ResponseWriter, r *http.Request) {
		var body setSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Checks.Password.Password != "hunter2" {
			t.Fatalf("password = %q", body.Checks.Password.Password)
		}
		_ = json.NewEncoder(w).Encode(setSessionResponse{SessionToken: "tok-2"})
	})
	service, sessions := newTestService(t, mux, nil)
	ctx := loadCtx(t, sessions)
	if _, err := service.startWithLoginName(ctx, "minnie@example.com"); err != nil {
		t.Fatal(err)
	}

	state, err := service.checkPassword(ctx, "hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if !state.PasswordVerified || !state.Sufficient() {
		t.Fatalf("state = %+v, want verified and sufficient", state)
	}
	if got := sessions.GetString(ctx, zitadelToken); got != "tok-2" {
		t.Fatalf("session token not rotated: got %q", got)
	}
}

func TestCheckPasswordRejectsWrongPassword(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	mux.HandleFunc("PATCH /v2/sessions/sess-1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	service, sessions := newTestService(t, mux, nil)
	ctx := loadCtx(t, sessions)
	if _, err := service.startWithLoginName(ctx, "minnie@example.com"); err != nil {
		t.Fatal(err)
	}

	if _, err := service.checkPassword(ctx, "wrong"); err == nil {
		t.Fatal("want error for wrong password")
	}
	if sessions.GetBool(ctx, passwordOKKey) {
		t.Fatal("password must not be marked verified on failure")
	}
}

func TestFinalizeRequiresSufficientState(t *testing.T) {
	t.Parallel()
	service, sessions := newTestService(t, http.NewServeMux(), &completerStub{})
	ctx := loadCtx(t, sessions)

	if _, err := service.finalize(ctx, "V2_authreq"); err == nil {
		t.Fatal("want error when session is not sufficient")
	}
}

func TestFinalizeCompletesAuthorizationCallbackAndClearsTransaction(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	mux.HandleFunc("PATCH /v2/sessions/sess-1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(setSessionResponse{SessionToken: "tok-2"})
	})
	mux.HandleFunc("POST /v2/oidc/auth_requests/V2_authreq", func(w http.ResponseWriter, r *http.Request) {
		var body createCallbackRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Session.SessionID != "sess-1" || body.Session.SessionToken != "tok-2" {
			t.Fatalf("unexpected session in callback request: %+v", body.Session)
		}
		_ = json.NewEncoder(w).Encode(createCallbackResponse{
			CallbackURL: "https://app.example/auth/callback?code=abc123&state=xyz",
		})
	})
	completer := &completerStub{redirectTo: "/dashboard"}
	service, sessions := newTestService(t, mux, completer)
	ctx := loadCtx(t, sessions)
	if _, err := service.startWithLoginName(ctx, "minnie@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.checkPassword(ctx, "hunter2"); err != nil {
		t.Fatal(err)
	}

	redirectTo, err := service.finalize(ctx, "V2_authreq")
	if err != nil {
		t.Fatal(err)
	}
	if redirectTo != "/dashboard" {
		t.Fatalf("redirectTo = %q, want /dashboard", redirectTo)
	}
	if completer.gotCode != "abc123" || completer.gotState != "xyz" {
		t.Fatalf("completer got code=%q state=%q", completer.gotCode, completer.gotState)
	}
	if sessions.Exists(ctx, zitadelSessionID) {
		t.Fatal("login transaction must be destroyed after finalize")
	}
}

func TestStartWithLoginNameSetsNeedsTOTPWhenAccountHasItEnrolled(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	mux.HandleFunc("POST /v2/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[{"userId":"user-1","human":{"email":{"email":"minnie@example.com"}}}]}`))
	})
	mux.HandleFunc("GET /v2/users/user-1/authentication_methods", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(authMethodsWithTOTPResponse))
	})
	service, sessions := newTestService(t, mux, nil)
	ctx := loadCtx(t, sessions)

	state, err := service.startWithLoginName(ctx, "minnie")
	if err != nil {
		t.Fatal(err)
	}
	if !state.NeedsTOTP {
		t.Fatal("want NeedsTOTP=true when the account has TOTP enrolled")
	}
	if state.Sufficient() {
		t.Fatal("must not be sufficient before password/TOTP are verified")
	}
}

func TestCheckTOTPRequiresPasswordFirst(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	service, sessions := newTestService(t, mux, nil)
	ctx := loadCtx(t, sessions)
	if _, err := service.startWithLoginName(ctx, "minnie"); err != nil {
		t.Fatal(err)
	}

	if _, err := service.checkTOTP(ctx, "123456"); err == nil {
		t.Fatal("want error when checking TOTP before password")
	}
}

func TestCheckTOTPSucceedsAfterPasswordAndMakesSessionSufficient(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	mux.HandleFunc("POST /v2/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[{"userId":"user-1","human":{"email":{"email":"minnie@example.com"}}}]}`))
	})
	mux.HandleFunc("GET /v2/users/user-1/authentication_methods", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(authMethodsWithTOTPResponse))
	})
	mux.HandleFunc("PATCH /v2/sessions/sess-1", func(w http.ResponseWriter, r *http.Request) {
		var body setSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch {
		case body.Checks.Password != nil:
			_ = json.NewEncoder(w).Encode(setSessionResponse{SessionToken: "tok-2"})
		case body.Checks.Totp != nil:
			if body.Checks.Totp.Code != "123456" {
				t.Fatalf("totp code = %q, want 123456", body.Checks.Totp.Code)
			}
			_ = json.NewEncoder(w).Encode(setSessionResponse{SessionToken: "tok-3"})
		default:
			t.Fatalf("unexpected checks: %+v", body.Checks)
		}
	})
	service, sessions := newTestService(t, mux, nil)
	ctx := loadCtx(t, sessions)
	if _, err := service.startWithLoginName(ctx, "minnie"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.checkPassword(ctx, "hunter2"); err != nil {
		t.Fatal(err)
	}

	state, err := service.checkTOTP(ctx, "123456")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Sufficient() {
		t.Fatalf("state = %+v, want sufficient", state)
	}
}
