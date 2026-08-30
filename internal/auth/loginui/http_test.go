package loginui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateSessionHandlerRejectsMissingLoginName(t *testing.T) {
	t.Parallel()
	service, sessions := newTestService(t, http.NewServeMux(), nil)
	handler := NewHandler(service)
	request := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader(`{}`))
	ctx := loadCtx(t, sessions)
	recorder := httptest.NewRecorder()

	handler.createSession(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestCreateSessionHandlerReturnsState(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	service, sessions := newTestService(t, mux, nil)
	handler := NewHandler(service)
	request := httptest.NewRequest(
		http.MethodPost, "/auth/session", strings.NewReader(`{"loginName":"minnie@example.com"}`),
	)
	ctx := loadCtx(t, sessions)
	recorder := httptest.NewRecorder()

	handler.createSession(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", recorder.Code, recorder.Body)
	}
	var state State
	if err := json.NewDecoder(recorder.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.LoginName != "minnie@example.com" || state.Sufficient() {
		t.Fatalf("state = %+v", state)
	}
}

func TestCreateSessionHandlerMapsInvalidCredentialsTo401(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	service, sessions := newTestService(t, mux, nil)
	handler := NewHandler(service)
	request := httptest.NewRequest(
		http.MethodPost, "/auth/session", strings.NewReader(`{"loginName":"nobody@example.com"}`),
	)
	ctx := loadCtx(t, sessions)
	recorder := httptest.NewRecorder()

	handler.createSession(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestCheckPasswordHandlerRejectsMissingPassword(t *testing.T) {
	t.Parallel()
	service, sessions := newTestService(t, http.NewServeMux(), nil)
	handler := NewHandler(service)
	request := httptest.NewRequest(http.MethodPatch, "/auth/session/password", strings.NewReader(`{}`))
	ctx := loadCtx(t, sessions)
	recorder := httptest.NewRecorder()

	handler.checkPassword(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestCheckPasswordHandlerWithoutActiveSessionReturnsConflict(t *testing.T) {
	t.Parallel()
	service, sessions := newTestService(t, http.NewServeMux(), nil)
	handler := NewHandler(service)
	request := httptest.NewRequest(
		http.MethodPatch, "/auth/session/password", strings.NewReader(`{"password":"hunter2"}`),
	)
	ctx := loadCtx(t, sessions)
	recorder := httptest.NewRecorder()

	handler.checkPassword(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", recorder.Code)
	}
}

func TestFinalizeHandlerRejectsMissingAuthRequestID(t *testing.T) {
	t.Parallel()
	service, sessions := newTestService(t, http.NewServeMux(), &completerStub{})
	handler := NewHandler(service)
	request := httptest.NewRequest(http.MethodPost, "/auth/session/finalize", strings.NewReader(`{}`))
	ctx := loadCtx(t, sessions)
	recorder := httptest.NewRecorder()

	handler.finalize(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestFinalizeHandlerReturnsRedirectURLOnSuccess(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createSessionResponse{SessionID: "sess-1", SessionToken: "tok-1"})
	})
	mux.HandleFunc("PATCH /v2/sessions/sess-1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(setSessionResponse{SessionToken: "tok-2"})
	})
	mux.HandleFunc("POST /v2/oidc/auth_requests/V2_authreq", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(createCallbackResponse{
			CallbackURL: "https://app.example/auth/callback?code=abc123&state=xyz",
		})
	})
	completer := &completerStub{redirectTo: "/dashboard"}
	service, sessions := newTestService(t, mux, completer)
	handler := NewHandler(service)
	ctx := loadCtx(t, sessions)
	if _, err := service.startWithLoginName(ctx, "minnie@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.checkPassword(ctx, "hunter2"); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodPost, "/auth/session/finalize", strings.NewReader(`{"authRequestId":"V2_authreq"}`),
	)
	recorder := httptest.NewRecorder()

	handler.finalize(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body)
	}
	var body finalizeResponseBody
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.RedirectURL != "/dashboard" {
		t.Fatalf("redirectUrl = %q, want /dashboard", body.RedirectURL)
	}
}
