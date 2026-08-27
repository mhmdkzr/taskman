package auditlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/app/pkg/middleware"
	"github.com/mhmdkzr/app/pkg/natsembed"
)

func TestEvent_MsgID(t *testing.T) {
	e := EventAPIAuditLogged{RequestID: "abc-123"}
	if id := e.MsgID(); id != "abc-123" {
		t.Fatalf("MsgID = %q, want %q", id, "abc-123")
	}
}

func TestEvent_MsgIDEmpty(t *testing.T) {
	e := EventAPIAuditLogged{}
	if id := e.MsgID(); id != "" {
		t.Fatalf("MsgID = %q, want empty", id)
	}
}

func TestEvent_JSONBodiesAreStrings(t *testing.T) {
	e := EventAPIAuditLogged{
		RequestID: "req-json",
		Request: middleware.RequestData{
			Body: []byte(`{"key":"val"}`),
		},
		Response: middleware.ResponseData{
			Body: []byte(`{"status":"ok"}`),
			Size: int64(len(`{"status":"ok"}`)),
		},
	}

	payload, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	assertRawBodyJSON(t, payload, "request", json.RawMessage(`{"key":"val"}`))
	assertRawBodyJSON(t, payload, "response", json.RawMessage(`{"status":"ok"}`))

	var decoded EventAPIAuditLogged
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if !bytes.Equal(decoded.Request.Body, e.Request.Body) {
		t.Fatalf("decoded request body = %q, want %q", string(decoded.Request.Body), string(e.Request.Body))
	}
	if !bytes.Equal(decoded.Response.Body, e.Response.Body) {
		t.Fatalf("decoded response body = %q, want %q", string(decoded.Response.Body), string(e.Response.Body))
	}
}

func TestNew_PublishesAuditEvent(t *testing.T) {
	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("nats embed connect: %v", err)
	}
	defer nc.Close()

	ctx := t.Context()
	subject, stream := uniqueAuditTestSubject(t)
	cfg := Config{Subject: subject, Stream: stream}
	if err := CreateStream(ctx, js, cfg); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	consumer, err := js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:          "test",
		FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}

	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mw, err := New(js, cfg)
	if err != nil {
		t.Fatalf("new middleware: %v", err)
	}
	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test?foo=bar", bytes.NewReader([]byte(`{"key":"val"}`)))
	r.RemoteAddr = "10.0.0.1:54321"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("User-Agent", "test-agent")

	mw(handler).ServeHTTP(rec, r)

	if !handlerCalled {
		t.Fatal("handler was not called")
	}

	msg, err := consumer.Fetch(1, jetstream.FetchMaxWait(time.Second))
	if err != nil {
		t.Fatalf("fetch audit message: %v", err)
	}

	var evt EventAPIAuditLogged
	var rawPayload []byte
	m, ok := <-msg.Messages()
	if !ok {
		t.Fatal("no messages received")
	}
	rawPayload = append([]byte(nil), m.Data()...)
	if err := json.Unmarshal(rawPayload, &evt); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	m.Ack()

	if evt.RequestID == "" {
		t.Fatal("request_id should not be empty")
	}
	if evt.Duration <= 0 {
		t.Fatal("duration should be positive")
	}
	if evt.Timestamp.IsZero() {
		t.Fatal("timestamp should be set")
	}

	if evt.Request.Method != http.MethodPost {
		t.Fatalf("method = %q, want %q", evt.Request.Method, "POST")
	}
	if evt.Request.Path != "/test" {
		t.Fatalf("path = %q, want %q", evt.Request.Path, "/test")
	}
	if evt.Request.Query != "foo=bar" {
		t.Fatalf("query = %q, want %q", evt.Request.Query, "foo=bar")
	}
	if evt.Request.RemoteAddr != "10.0.0.1:54321" {
		t.Fatalf("remote addr = %q, want %q", evt.Request.RemoteAddr, "10.0.0.1:54321")
	}
	assertRawBodyJSON(t, rawPayload, "request", json.RawMessage(`{"key":"val"}`))
	if string(evt.Request.Body) != `{"key":"val"}` {
		t.Fatalf("request body = %q, want %q", string(evt.Request.Body), `{"key":"val"}`)
	}

	if evt.Response.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", evt.Response.StatusCode, http.StatusOK)
	}
	assertRawBodyJSON(t, rawPayload, "response", json.RawMessage(`{"status":"ok"}`))
	if string(evt.Response.Body) != `{"status":"ok"}` {
		t.Fatalf("response body = %q, want %q", string(evt.Response.Body), `{"status":"ok"}`)
	}
}

func TestNew_WithoutBody(t *testing.T) {
	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("nats embed connect: %v", err)
	}
	defer nc.Close()

	ctx := t.Context()
	subject, stream := uniqueAuditTestSubject(t)
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       stream,
		Subjects:   []string{subject},
		Duplicates: time.Hour,
	}); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	consumer, err := js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:          "test-no-body",
		FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mw := newWithSubject(js, subject, nil)
	mw(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	msg, err := consumer.Fetch(1, jetstream.FetchMaxWait(time.Second))
	if err != nil {
		t.Fatalf("fetch audit message: %v", err)
	}

	var evt EventAPIAuditLogged
	m, ok := <-msg.Messages()
	if !ok {
		t.Fatal("no messages received")
	}
	if err := json.Unmarshal(m.Data(), &evt); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	m.Ack()

	if evt.Request.Method != http.MethodGet {
		t.Fatalf("method = %q, want %q", evt.Request.Method, "GET")
	}
}

func TestNew_HandlerBodyStillReadable(t *testing.T) {
	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("nats embed connect: %v", err)
	}
	defer nc.Close()

	mw := mustNew(t, js, Config{Subject: "test.audit", Stream: "TEST_AUDIT"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		if buf.String() != "hello" {
			t.Fatalf("handler reads body = %q, want %q", buf.String(), "hello")
		}
		w.WriteHeader(http.StatusOK)
	})

	mw(handler).ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("hello"))))
}

func TestNewWithRedactor_PublishesRedactedAuditEvent(t *testing.T) {
	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("nats embed connect: %v", err)
	}
	defer nc.Close()

	ctx := t.Context()
	subject, stream := uniqueAuditTestSubject(t)
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       stream,
		Subjects:   []string{subject},
		Duplicates: time.Hour,
	}); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	consumer, err := js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:          "test-redacted",
		FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}

	redactor := func(event EventAPIAuditLogged) EventAPIAuditLogged {
		event.Request.Body = []byte(`{"password":"[REDACTED]"}`)
		event.Request.Headers.Set("Authorization", "[REDACTED]")
		return event
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mw := newWithSubject(js, subject, redactor)
	r := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(`{"password":"secret"}`)))
	r.Header.Set("Authorization", "Bearer secret")

	mw(handler).ServeHTTP(httptest.NewRecorder(), r)

	msg, err := consumer.Fetch(1, jetstream.FetchMaxWait(time.Second))
	if err != nil {
		t.Fatalf("fetch audit message: %v", err)
	}

	var evt EventAPIAuditLogged
	m, ok := <-msg.Messages()
	if !ok {
		t.Fatal("no messages received")
	}
	if err := json.Unmarshal(m.Data(), &evt); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	m.Ack()

	if evt.Request.Headers.Get("Authorization") != "[REDACTED]" {
		t.Fatalf("authorization = %q, want redacted", evt.Request.Headers.Get("Authorization"))
	}
	if string(evt.Request.Body) != `{"password":"[REDACTED]"}` {
		t.Fatalf("body = %q, want redacted", string(evt.Request.Body))
	}
}

func TestNew_SetsRequestID(t *testing.T) {
	nc, js, err := natsembed.Connect()
	if err != nil {
		t.Fatalf("nats embed connect: %v", err)
	}
	defer nc.Close()

	t.Run("generates when header missing", func(t *testing.T) {
		var gotID string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotID = middleware.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		mw := mustNew(t, js, Config{Subject: "test.audit", Stream: "TEST_AUDIT"})
		rec := httptest.NewRecorder()
		mw(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if gotID == "" {
			t.Fatal("request_id should be set in context")
		}
		if rec.Header().Get("X-Request-ID") == "" {
			t.Fatal("X-Request-ID response header should be set")
		}
	})

	t.Run("accepts header when present", func(t *testing.T) {
		var gotID string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotID = middleware.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		mw := mustNew(t, js, Config{Subject: "test.audit", Stream: "TEST_AUDIT"})
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Request-ID", "my-trace-id")
		mw(handler).ServeHTTP(rec, r)

		if gotID != "my-trace-id" {
			t.Fatalf("request_id = %q, want %q", gotID, "my-trace-id")
		}
		if rec.Header().Get("X-Request-ID") != "my-trace-id" {
			t.Fatalf("X-Request-ID response header = %q, want %q", rec.Header().Get("X-Request-ID"), "my-trace-id")
		}
	})
}

func assertRawBodyJSON(t *testing.T, payload []byte, section string, want json.RawMessage) {
	t.Helper()

	var event map[string]json.RawMessage
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal raw event: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(event[section], &fields); err != nil {
		t.Fatalf("unmarshal raw %s: %v", section, err)
	}

	got := fields["body"]
	var gotNorm, wantNorm any
	if err := json.Unmarshal(got, &gotNorm); err != nil {
		t.Fatalf("unmarshal raw %s body: %v", section, err)
	}
	if err := json.Unmarshal(want, &wantNorm); err != nil {
		t.Fatalf("unmarshal expected body: %v", err)
	}
	gotJSON, err := json.Marshal(gotNorm)
	if err != nil {
		t.Fatalf("marshal raw %s body: %v", section, err)
	}
	wantJSON, err := json.Marshal(wantNorm)
	if err != nil {
		t.Fatalf("marshal expected body: %v", err)
	}
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("raw %s body = %s, want %s", section, string(gotJSON), string(wantJSON))
	}
}

func uniqueAuditTestSubject(t *testing.T) (subject, stream string) {
	t.Helper()

	suffix := fmt.Sprintf("%s_%d", t.Name(), time.Now().UnixNano())
	return "api.audit." + suffix, "API_AUDIT_" + suffix
}

func mustNew(t *testing.T, js jetstream.JetStream, cfg Config) middleware.Middleware {
	t.Helper()

	mw, err := New(js, cfg)
	if err != nil {
		t.Fatalf("new audit middleware: %v", err)
	}
	return mw
}
