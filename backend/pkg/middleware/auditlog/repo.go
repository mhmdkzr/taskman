package auditlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// insertAuditLog inserts an audit event into the database.
func insertAuditLog(ctx context.Context, db *sql.DB, event EventAPIAuditLogged) error {
	requestHeaders, err := json.Marshal(event.Request.Headers)
	if err != nil {
		return fmt.Errorf("marshal request headers: %w", err)
	}
	responseHeaders, err := json.Marshal(event.Response.Headers)
	if err != nil {
		return fmt.Errorf("marshal response headers: %w", err)
	}

	requestBody := toJSONBCompatibleString(event.Request.Body)
	responseBody := toJSONBCompatibleString(event.Response.Body)

	_, err = db.ExecContext(ctx, `
		INSERT INTO core.audit_logs (
			request_id,
			timestamp,
			duration_ns,
			request_method,
			request_path,
			request_query,
			request_remote_addr,
			request_user_agent,
			request_proto,
			request_headers,
			request_body,
			response_status_code,
			response_headers,
			response_body,
			response_size
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, $13::jsonb, $14::jsonb, $15)
		ON CONFLICT (request_id) DO NOTHING`,
		event.RequestID,
		event.Timestamp,
		event.Duration.Nanoseconds(),
		event.Request.Method,
		event.Request.Path,
		event.Request.Query,
		event.Request.RemoteAddr,
		event.Request.UserAgent,
		event.Request.Proto,
		requestHeaders,
		requestBody,
		event.Response.StatusCode,
		responseHeaders,
		responseBody,
		event.Response.Size,
	)
	if err != nil {
		return fmt.Errorf("insert audit log %s: %w", event.RequestID, err)
	}
	return nil
}

// toJSONBCompatibleString converts a byte slice to a JSONB-compatible string pointer.
func toJSONBCompatibleString(body []byte) *string {
	if len(body) == 0 {
		return nil
	}
	if json.Valid(body) {
		s := string(body)
		return &s
	}
	b, err := json.Marshal(string(body))
	if err != nil {
		s := string(body)
		return &s
	}
	s := string(b)
	return &s
}
