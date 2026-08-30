package loginui

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// recordingMailer records every Send call, standing in for a real SMTP
// client so tests can assert on the email content without sending mail.
type recordingMailer struct {
	mu   sync.Mutex
	sent []sentMail
	err  error
}

type sentMail struct {
	to, subject, body string
}

func (m *recordingMailer) Send(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, sentMail{to: to, subject: subject, body: body})
	return nil
}

func (m *recordingMailer) last() sentMail {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.sent) == 0 {
		return sentMail{}
	}
	return m.sent[len(m.sent)-1]
}

// newRegistrationTestService builds a Service (bypassing New's transaction
// setup, which registration doesn't use) wired to a stub ZITADEL REST server
// and a recordingMailer.
func newRegistrationTestService(t *testing.T, mux *http.ServeMux, mailer *recordingMailer) *Service {
	t.Helper()
	service, _ := newTestService(t, mux, nil)
	service.mailer = mailer
	service.frontendBaseURL = "https://app.example"
	return service
}

func TestRegisterSendsVerificationEmailWithCode(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/users/human", func(w http.ResponseWriter, r *http.Request) {
		var body registerHumanRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Username != "minnie" || body.Email.Email != "minnie@example.com" {
			t.Fatalf("unexpected request: %+v", body)
		}
		_, _ = w.Write([]byte(`{"userId":"user-1","emailCode":"CODE123"}`))
	})
	mailer := &recordingMailer{}
	service := newRegistrationTestService(t, mux, mailer)

	userID, err := service.Register(t.Context(), "minnie", "minnie@example.com", "hunter2", "Minnie", "Mouse")
	if err != nil {
		t.Fatal(err)
	}
	if userID != "user-1" {
		t.Fatalf("userID = %q, want user-1", userID)
	}
	sent := mailer.last()
	if sent.to != "minnie@example.com" {
		t.Fatalf("sent to %q, want minnie@example.com", sent.to)
	}
	if !containsAll(sent.body, "https://app.example/verify-email", "userId=user-1", "code=CODE123") {
		t.Fatalf("email body missing verification link: %q", sent.body)
	}
}

func TestRegisterPropagatesZitadelError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/users/human", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	mailer := &recordingMailer{}
	service := newRegistrationTestService(t, mux, mailer)

	_, err := service.Register(t.Context(), "minnie", "minnie@example.com", "hunter2", "Minnie", "Mouse")
	if err == nil {
		t.Fatal("want error")
	}
	if len(mailer.sent) != 0 {
		t.Fatal("must not send an email when registration fails")
	}
}

func TestVerifyEmailCallsZitadel(t *testing.T) {
	t.Parallel()
	var gotCode string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/users/user-1/email/verify", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		gotCode = body["verificationCode"]
	})
	service := newRegistrationTestService(t, mux, &recordingMailer{})

	if err := service.VerifyEmail(t.Context(), "user-1", "CODE123"); err != nil {
		t.Fatal(err)
	}
	if gotCode != "CODE123" {
		t.Fatalf("verificationCode = %q, want CODE123", gotCode)
	}
}

func TestRequestPasswordResetSendsEmailWhenUserExists(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[{"userId":"user-1","human":{"email":{"email":"minnie@example.com"}}}]}`))
	})
	mux.HandleFunc("POST /v2/users/user-1/password_reset", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"verificationCode":"RESETCODE"}`))
	})
	mailer := &recordingMailer{}
	service := newRegistrationTestService(t, mux, mailer)

	if err := service.RequestPasswordReset(t.Context(), "minnie"); err != nil {
		t.Fatal(err)
	}
	sent := mailer.last()
	if sent.to != "minnie@example.com" {
		t.Fatalf("sent to %q, want minnie@example.com", sent.to)
	}
	if !containsAll(sent.body, "https://app.example/reset-password", "userId=user-1", "code=RESETCODE") {
		t.Fatalf("email body missing reset link: %q", sent.body)
	}
}

func TestRequestPasswordResetSucceedsSilentlyForUnknownUser(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[]}`))
	})
	mailer := &recordingMailer{}
	service := newRegistrationTestService(t, mux, mailer)

	if err := service.RequestPasswordReset(t.Context(), "nobody"); err != nil {
		t.Fatalf("want no error for unknown login name, got %v", err)
	}
	if len(mailer.sent) != 0 {
		t.Fatal("must not send an email for an unknown login name")
	}
}

func TestResetPasswordCallsZitadel(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/users/user-1/password", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
	})
	service := newRegistrationTestService(t, mux, &recordingMailer{})

	if err := service.ResetPassword(t.Context(), "user-1", "RESETCODE", "NewPass123!"); err != nil {
		t.Fatal(err)
	}
	if gotBody["verificationCode"] != "RESETCODE" {
		t.Fatalf("verificationCode = %v, want RESETCODE", gotBody["verificationCode"])
	}
}

func TestTOTPEnrollmentStartAndConfirm(t *testing.T) {
	t.Parallel()
	var confirmedCode string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v2/users/user-1/totp", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"uri":"otpauth://totp/example","secret":"SECRET123"}`))
	})
	mux.HandleFunc("POST /v2/users/user-1/totp/verify", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		confirmedCode = body["code"]
	})
	service := newRegistrationTestService(t, mux, &recordingMailer{})

	uri, secret, err := service.StartTOTPEnrollment(t.Context(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if uri != "otpauth://totp/example" || secret != "SECRET123" {
		t.Fatalf("uri=%q secret=%q", uri, secret)
	}
	if err := service.ConfirmTOTPEnrollment(t.Context(), "user-1", "123456"); err != nil {
		t.Fatal(err)
	}
	if confirmedCode != "123456" {
		t.Fatalf("confirmedCode = %q, want 123456", confirmedCode)
	}
}

func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
