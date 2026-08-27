package jsonresp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSONAndError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSON(rr, http.StatusCreated, map[string]string{"status": "ok"})

	if rr.Code != http.StatusCreated {
		t.Fatalf("status mismatch: got=%d want=%d", rr.Code, http.StatusCreated)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type mismatch: got=%q", ct)
	}

	var got map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("payload mismatch: %#v", got)
	}

	rr = httptest.NewRecorder()
	WriteHTTPError(rr, http.StatusBadRequest, errors.New("boom"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got=%d want=%d", rr.Code, http.StatusBadRequest)
	}
	if body := rr.Body.String(); !strings.Contains(body, `"error":"boom"`) {
		t.Fatalf("unexpected error body: %s", body)
	}
}

func TestWriteJSON_EncodeError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSON(rr, http.StatusOK, map[string]chan int{"bad": make(chan int)})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status mismatch: got=%d want=%d", rr.Code, http.StatusInternalServerError)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type mismatch: got=%q", ct)
	}
	if body := rr.Body.String(); !strings.Contains(body, `"error":"failed to encode json response"`) {
		t.Fatalf("unexpected fallback body: %s", body)
	}
}
