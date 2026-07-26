package auditlog

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/mhmdkzr/app/pkg/middleware"
	coretestdb "github.com/mhmdkzr/app/pkg/testdb"
	"github.com/mhmdkzr/app/pkg/testenv"
)

func TestInsertAuditLog_PersistsFullEvent(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	t.Parallel()

	db := coretestdb.SetupTestPostgres(t, "audit_log_insert")
	event := testAuditEvent("req-full")

	if err := insertAuditLog(context.Background(), db, event); err != nil {
		t.Fatalf("insertAuditLog: %v", err)
	}

	var (
		timestamp           time.Time
		durationNS          int64
		method              string
		path                string
		query               string
		remoteAddr          string
		userAgent           string
		proto               string
		requestContentType  string
		requestTrace        string
		requestBody         []byte
		statusCode          int
		responseContentType string
		responseBody        []byte
		responseSize        int64
	)
	if err := db.QueryRow(`
		SELECT
			timestamp,
			duration_ns,
			request_method,
			request_path,
			request_query,
			request_remote_addr,
			request_user_agent,
			request_proto,
			request_headers->'Content-Type'->>0,
			request_headers->'X-Trace'->>0,
			request_body,
			response_status_code,
			response_headers->'Content-Type'->>0,
			response_body,
			response_size
		FROM core.audit_logs
		WHERE request_id = $1`, event.RequestID).Scan(
		&timestamp,
		&durationNS,
		&method,
		&path,
		&query,
		&remoteAddr,
		&userAgent,
		&proto,
		&requestContentType,
		&requestTrace,
		&requestBody,
		&statusCode,
		&responseContentType,
		&responseBody,
		&responseSize,
	); err != nil {
		t.Fatalf("query audit log: %v", err)
	}

	if !timestamp.Equal(event.Timestamp) {
		t.Fatalf("timestamp = %s, want %s", timestamp, event.Timestamp)
	}
	if durationNS != event.Duration.Nanoseconds() {
		t.Fatalf("duration_ns = %d, want %d", durationNS, event.Duration.Nanoseconds())
	}
	if method != event.Request.Method || path != event.Request.Path || query != event.Request.Query {
		t.Fatalf(
			"request route = %s %s?%s, want %s %s?%s",
			method,
			path,
			query,
			event.Request.Method,
			event.Request.Path,
			event.Request.Query,
		)
	}
	if remoteAddr != event.Request.RemoteAddr || userAgent != event.Request.UserAgent || proto != event.Request.Proto {
		t.Fatalf("request metadata mismatch")
	}
	if requestContentType != "application/json" || requestTrace != "trace-1" {
		t.Fatalf("request headers = content-type %q trace %q", requestContentType, requestTrace)
	}
	if !bytes.Equal(requestBody, event.Request.Body) {
		t.Fatalf("request body = %q, want %q", string(requestBody), string(event.Request.Body))
	}
	if statusCode != event.Response.StatusCode {
		t.Fatalf("status code = %d, want %d", statusCode, event.Response.StatusCode)
	}
	if responseContentType != "application/json" {
		t.Fatalf("response content-type = %q", responseContentType)
	}
	if !bytes.Equal(responseBody, event.Response.Body) {
		t.Fatalf("response body = %q, want %q", string(responseBody), string(event.Response.Body))
	}
	if responseSize != event.Response.Size {
		t.Fatalf("response size = %d, want %d", responseSize, event.Response.Size)
	}
}

func TestInsertAuditLog_DuplicateRequestIDIsIdempotent(t *testing.T) {
	testenv.SkipIfDBTestsDisabled(t)
	t.Parallel()

	db := coretestdb.SetupTestPostgres(t, "audit_log_dedup")
	event := testAuditEvent("req-dedup")
	if err := insertAuditLog(context.Background(), db, event); err != nil {
		t.Fatalf("insertAuditLog first: %v", err)
	}

	duplicate := event
	duplicate.Request.Path = "/changed"
	if err := insertAuditLog(context.Background(), db, duplicate); err != nil {
		t.Fatalf("insertAuditLog duplicate: %v", err)
	}

	var count int
	var path string
	if err := db.QueryRow(`SELECT COUNT(1), MAX(request_path) FROM core.audit_logs WHERE request_id = $1`, event.RequestID).
		Scan(&count, &path); err != nil {
		t.Fatalf("query duplicate audit log: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if path != event.Request.Path {
		t.Fatalf("path = %q, want original %q", path, event.Request.Path)
	}
}

func testAuditEvent(requestID string) EventAPIAuditLogged {
	return EventAPIAuditLogged{
		RequestID: requestID,
		Timestamp: time.Date(2026, 6, 13, 10, 11, 12, 0, time.UTC),
		Duration:  25 * time.Millisecond,
		Request: middlewareauditlogRequestData(
			requestID,
			"POST",
			"/resources",
			"dry_run=true",
			[]byte(`{"resource_id":"resource-1"}`),
		),
		Response: middlewareauditlogResponseData(
			http.StatusCreated,
			[]byte(`{"status":"created"}`),
		),
	}
}

func middlewareauditlogRequestData(requestID, method, path, query string, body []byte) middleware.RequestData {
	return middleware.RequestData{
		ID:         requestID,
		Method:     method,
		Path:       path,
		Query:      query,
		RemoteAddr: "10.0.0.1:1000",
		UserAgent:  "audit-test",
		Proto:      "HTTP/1.1",
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Trace":      []string{"trace-1"},
		},
		Body: body,
	}
}

func middlewareauditlogResponseData(statusCode int, body []byte) middleware.ResponseData {
	return middleware.ResponseData{
		StatusCode: statusCode,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: body,
		Size: int64(len(body)),
	}
}
