package auditlog

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/mhmdkzr/app/pkg/middleware"
)

func TestAuditEvent_RedactsHeadersWithoutMutatingOriginal(t *testing.T) {
	event := EventAPIAuditLogged{
		Request: middleware.RequestData{
			Headers: http.Header{
				"Authorization": []string{"Bearer secret"},
				"Content-Type":  []string{"application/json"},
			},
		},
		Response: middleware.ResponseData{
			Headers: http.Header{
				"Set-Cookie": []string{"session=secret"},
				"X-Trace":    []string{"trace-1"},
			},
		},
	}

	got := AuditEvent(event)

	if got.Request.Headers.Get("Authorization") != Marker {
		t.Fatalf("authorization header = %q, want marker", got.Request.Headers.Get("Authorization"))
	}
	if got.Response.Headers.Get("Set-Cookie") != Marker {
		t.Fatalf("set-cookie header = %q, want marker", got.Response.Headers.Get("Set-Cookie"))
	}
	if got.Request.Headers.Get("Content-Type") != "application/json" {
		t.Fatalf("content-type header changed: %q", got.Request.Headers.Get("Content-Type"))
	}
	if event.Request.Headers.Get("Authorization") != "Bearer secret" {
		t.Fatalf("original request header mutated: %q", event.Request.Headers.Get("Authorization"))
	}
	if event.Response.Headers.Get("Set-Cookie") != "session=secret" {
		t.Fatalf("original response header mutated: %q", event.Response.Headers.Get("Set-Cookie"))
	}
}

func TestAuditEvent_RedactsSecretQueryParameters(t *testing.T) {
	event := EventAPIAuditLogged{
		Request: middleware.RequestData{
			Query: "token=abc&scope=read&api_key=key-1&foo=bar",
		},
	}

	got := AuditEvent(event)
	values, err := url.ParseQuery(got.Request.Query)
	if err != nil {
		t.Fatalf("parse redacted query: %v", err)
	}

	if values.Get("token") != Marker {
		t.Fatalf("token = %q, want marker", values.Get("token"))
	}
	if values.Get("api_key") != Marker {
		t.Fatalf("api_key = %q, want marker", values.Get("api_key"))
	}
	if values.Get("scope") != "read" {
		t.Fatalf("scope = %q, want read", values.Get("scope"))
	}
	if values.Get("foo") != "bar" {
		t.Fatalf("foo = %q, want bar", values.Get("foo"))
	}
}

func TestAuditEvent_RedactsOAuthCallbackQueryParameters(t *testing.T) {
	event := EventAPIAuditLogged{
		Request: middleware.RequestData{
			Query: "code=authorization-code&state=callback-state&error_description=keep",
		},
	}

	got := AuditEvent(event)
	values, err := url.ParseQuery(got.Request.Query)
	if err != nil {
		t.Fatalf("parse redacted query: %v", err)
	}
	for _, key := range []string{"code", "state"} {
		if values.Get(key) != Marker {
			t.Fatalf("%s = %q, want marker", key, values.Get(key))
		}
	}
	if values.Get("error_description") != "keep" {
		t.Fatalf("error_description = %q, want keep", values.Get("error_description"))
	}
}

func TestAuditEvent_RedactsRecursiveJSONBodies(t *testing.T) {
	event := EventAPIAuditLogged{
		Request: middleware.RequestData{
			Body: []byte(
				`{"password":"p","nested":{"access_token":"a","scope":"read"},"items":[{"private-key":"k"},{"memo":"keep"}]}`,
			),
		},
		Response: middleware.ResponseData{
			Body: []byte(`{"refreshToken":"r","user_id":"user-1"}`),
		},
	}

	got := AuditEvent(event)

	var request map[string]any
	if err := json.Unmarshal(got.Request.Body, &request); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	nested := request["nested"].(map[string]any)
	items := request["items"].([]any)
	firstItem := items[0].(map[string]any)
	secondItem := items[1].(map[string]any)

	if request["password"] != Marker {
		t.Fatalf("password = %#v, want marker", request["password"])
	}
	if nested["access_token"] != Marker {
		t.Fatalf("access_token = %#v, want marker", nested["access_token"])
	}
	if nested["scope"] != "read" {
		t.Fatalf("scope = %#v, want read", nested["scope"])
	}
	if firstItem["private-key"] != Marker {
		t.Fatalf("private-key = %#v, want marker", firstItem["private-key"])
	}
	if secondItem["memo"] != "keep" {
		t.Fatalf("memo = %#v, want keep", secondItem["memo"])
	}

	var response map[string]any
	if err := json.Unmarshal(got.Response.Body, &response); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if response["refreshToken"] != Marker {
		t.Fatalf("refreshToken = %#v, want marker", response["refreshToken"])
	}
	if response["user_id"] != "user-1" {
		t.Fatalf("user_id = %#v, want user-1", response["user_id"])
	}
}

func TestAuditEvent_PreservesNonJSONBodies(t *testing.T) {
	event := EventAPIAuditLogged{
		Request:  middleware.RequestData{Body: []byte("not-json token=abc")},
		Response: middleware.ResponseData{Body: []byte("{broken")},
	}

	got := AuditEvent(event)

	if string(got.Request.Body) != "not-json token=abc" {
		t.Fatalf("request body = %q, want unchanged", string(got.Request.Body))
	}
	if string(got.Response.Body) != "{broken" {
		t.Fatalf("response body = %q, want unchanged", string(got.Response.Body))
	}
}

func TestAuditEvent_PreservesOperationalFields(t *testing.T) {
	event := EventAPIAuditLogged{
		Request: middleware.RequestData{
			Body: []byte(
				`{"request_id":"req-1","memo":"note","user_id":"user-1","email":"user@example.com"}`,
			),
		},
	}

	got := AuditEvent(event)

	var body map[string]any
	if err := json.Unmarshal(got.Request.Body, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	for key, want := range map[string]string{
		"request_id": "req-1",
		"memo":       "note",
		"user_id":    "user-1",
		"email":      "user@example.com",
	} {
		if body[key] != want {
			t.Fatalf("%s = %#v, want %q", key, body[key], want)
		}
	}
}

func TestAuditEventInPlace_NilIsSafe(t *testing.T) {
	AuditEventInPlace(nil)
}
