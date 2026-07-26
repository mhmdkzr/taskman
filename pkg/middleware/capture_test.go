package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResponseRecorder_DefaultStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	rec := NewResponseRecorder(w)

	if rec.StatusCode != http.StatusOK {
		t.Fatalf("default status = %d, want %d", rec.StatusCode, http.StatusOK)
	}
}

func TestResponseRecorder_WritesStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	rec := NewResponseRecorder(w)

	rec.WriteHeader(http.StatusNotFound)

	if rec.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.StatusCode, http.StatusNotFound)
	}
}

func TestResponseRecorder_PreservesFirstStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	rec := NewResponseRecorder(w)

	rec.WriteHeader(http.StatusCreated)
	rec.WriteHeader(http.StatusInternalServerError)

	if rec.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.StatusCode, http.StatusCreated)
	}
	if w.Code != http.StatusCreated {
		t.Fatalf("underlying status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestResponseRecorder_CapturesBody(t *testing.T) {
	w := httptest.NewRecorder()
	rec := NewResponseRecorder(w)

	body := []byte("hello world")
	n, err := rec.Write(body)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	if n != len(body) {
		t.Fatalf("written = %d, want %d", n, len(body))
	}
	if !bytes.Equal(rec.Body.Bytes(), body) {
		t.Fatalf("body = %q, want %q", rec.Body.String(), string(body))
	}
	if rec.BytesWritten != int64(len(body)) {
		t.Fatalf("bytes written = %d, want %d", rec.BytesWritten, len(body))
	}
}

func TestResponseRecorder_WritesPassThrough(t *testing.T) {
	w := httptest.NewRecorder()
	rec := NewResponseRecorder(w)

	body := []byte("pass through")
	rec.Write(body)

	if w.Body.String() != string(body) {
		t.Fatalf("underlying response = %q, want %q", w.Body.String(), string(body))
	}
}

func TestResponseRecorder_HeaderPassthrough(t *testing.T) {
	w := httptest.NewRecorder()
	rec := NewResponseRecorder(w)

	rec.Header().Set("X-Custom", "value")
	rec.WriteHeader(http.StatusTeapot)

	if w.Code != http.StatusTeapot {
		t.Fatalf("underlying status = %d, want %d", w.Code, http.StatusTeapot)
	}
	if w.Header().Get("X-Custom") != "value" {
		t.Fatalf("underlying header missing")
	}
}

func TestContextHelpers_RequestID(t *testing.T) {
	ctx := context.Background()
	if id := RequestIDFromContext(ctx); id != "" {
		t.Fatalf("unexpected id from empty context: %q", id)
	}

	ctx = WithRequestID(ctx, "abc-123")
	if id := RequestIDFromContext(ctx); id != "abc-123" {
		t.Fatalf("id = %q, want %q", id, "abc-123")
	}
}

func TestContextHelpers_StartTime(t *testing.T) {
	ctx := context.Background()
	if _, ok := StartTimeFromContext(ctx); ok {
		t.Fatal("start time should not be present in empty context")
	}

	now := time.Now().Truncate(time.Millisecond)
	ctx = WithStartTime(ctx, now)
	got, ok := StartTimeFromContext(ctx)
	if !ok {
		t.Fatal("start time should be present")
	}
	if !got.Equal(now) {
		t.Fatalf("start time = %v, want %v", got, now)
	}
}

func TestContextHelpers_ClientIP(t *testing.T) {
	ctx := context.Background()
	if ip := ClientIPFromContext(ctx); ip != "" {
		t.Fatalf("unexpected ip from empty context: %q", ip)
	}

	ctx = WithClientIP(ctx, "10.0.0.1")
	if ip := ClientIPFromContext(ctx); ip != "10.0.0.1" {
		t.Fatalf("ip = %q, want %q", ip, "10.0.0.1")
	}
}

func TestRecorderInRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := RecorderFromContext(r); ok {
		t.Fatal("recorder should not be present in empty request")
	}

	w := httptest.NewRecorder()
	rec := NewResponseRecorder(w)
	r = WithRecorder(r, rec)

	got, ok := RecorderFromContext(r)
	if !ok {
		t.Fatal("recorder should be present")
	}
	if got != rec {
		t.Fatal("recorder pointer mismatch")
	}
}

func TestRequestData_JSONTags(t *testing.T) {
	d := RequestData{
		ID:         "r1",
		Method:     "POST",
		Path:       "/test",
		Query:      "foo=bar",
		RemoteAddr: "10.0.0.1:8080",
		UserAgent:  "test-agent",
		Proto:      "HTTP/1.1",
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"key":"val"}`),
	}

	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"id", "method", "path", "query", "remote_addr", "user_agent", "proto", "headers", "body"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("RequestData missing JSON key %q", key)
		}
	}
	bodyMap, ok := m["body"].(map[string]any)
	if !ok || bodyMap["key"] != "val" {
		t.Fatalf("body = %#v, want JSON object body", m["body"])
	}

	var decoded RequestData
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal request data: %v", err)
	}
	if !bytes.Equal(decoded.Body, d.Body) {
		t.Fatalf("decoded body = %q, want %q", string(decoded.Body), string(d.Body))
	}
}

func TestResponseData_JSONTags(t *testing.T) {
	d := ResponseData{
		StatusCode: 200,
		Headers:    http.Header{"X-Custom": []string{"val"}},
		Body:       []byte("ok"),
		Size:       2,
	}

	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"status_code", "headers", "body", "size"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("ResponseData missing JSON key %q", key)
		}
	}
	if m["body"] != "ok" {
		t.Fatalf("body = %#v, want plain string body", m["body"])
	}

	var decoded ResponseData
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal response data: %v", err)
	}
	if !bytes.Equal(decoded.Body, d.Body) {
		t.Fatalf("decoded body = %q, want %q", string(decoded.Body), string(d.Body))
	}
}
