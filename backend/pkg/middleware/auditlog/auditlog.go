package auditlog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mhmdkzr/app/pkg/middleware"
)

// SubjectAPIAudit is the NATS subject for API audit events.
const (
	SubjectAPIAudit = "api.audit"
	publishTimeout  = 5 * time.Second
)

// New returns a middleware that logs API audit events to JetStream.
func New(js jetstream.JetStream) middleware.Middleware {
	return NewWithRedactor(js, nil)
}

// NewWithRedactor returns an audit logging middleware with a redaction function.
func NewWithRedactor(
	js jetstream.JetStream,
	redactor func(EventAPIAuditLogged) EventAPIAuditLogged,
) middleware.Middleware {
	return newWithSubject(js, SubjectAPIAudit, redactor)
}

// newWithSubject returns an audit logging middleware that publishes to the given subject.
func newWithSubject(
	js jetstream.JetStream,
	subject string,
	redactor func(EventAPIAuditLogged) EventAPIAuditLogged,
) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				slog.Debug("request missing X-Request-ID header, generating")
				id, err := uuid.NewV7()
				if err != nil {
					slog.Error("failed to generate uuid v7, falling back to v4", "error", err)
					requestID = uuid.NewString()
				} else {
					requestID = id.String()
				}
			}
			w.Header().Set("X-Request-ID", requestID)

			ctx := middleware.WithRequestID(r.Context(), requestID)
			ctx = middleware.WithStartTime(ctx, time.Now())
			r = r.WithContext(ctx)

			reqBody, reqTruncated, r2 := drainBody(r)
			r = r2

			rec := middleware.NewResponseRecorder(w)
			r = middleware.WithRecorder(r, rec)

			t0 := time.Now()
			next.ServeHTTP(rec, r)
			duration := time.Since(t0)

			event := EventAPIAuditLogged{
				RequestID: requestID,
				Timestamp: t0,
				Duration:  duration,
				Request: middleware.RequestData{
					ID:         requestID,
					Method:     r.Method,
					Path:       r.URL.Path,
					Query:      r.URL.RawQuery,
					RemoteAddr: r.RemoteAddr,
					UserAgent:  r.UserAgent(),
					Proto:      r.Proto,
					Headers:    r.Header,
					Body:       reqBody,
					Truncated:  reqTruncated,
				},
				Response: middleware.ResponseData{
					StatusCode: rec.StatusCode,
					Headers:    rec.Header(),
					Body:       rec.Body.Bytes(),
					Size:       rec.BytesWritten,
					Truncated:  rec.Truncated,
				},
			}

			if redactor != nil {
				event = redactor(event)
			}

			publishCtx, cancel := context.WithTimeout(r.Context(), publishTimeout)
			defer cancel()
			//nolint:contextcheck // publishCtx is derived from r.Context with a shorter audit publish timeout.
			if err := publish(publishCtx, js, subject, event); err != nil {
				slog.Error("failed to publish audit event", "error", err, "request_id", requestID)
			}
		})
	}
}

// drainBody reads the request body for audit logging while preserving it for handlers.
func drainBody(r *http.Request) ([]byte, bool, *http.Request) {
	if r.Body == nil {
		return nil, false, r
	}
	var buf bytes.Buffer
	limited := io.LimitReader(r.Body, middleware.DefaultCaptureBodyLimit+1)
	_, err := buf.ReadFrom(limited)
	if err != nil {
		slog.Error("failed to read request body for audit", "error", err)
		return nil, false, r
	}
	consumed := buf.Bytes()
	b := consumed
	truncated := int64(len(consumed)) > middleware.DefaultCaptureBodyLimit
	if truncated {
		b = consumed[:middleware.DefaultCaptureBodyLimit]
	}
	r.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: io.MultiReader(bytes.NewReader(consumed), r.Body),
		Closer: r.Body,
	}
	return b, truncated, r
}

// publish publishes an audit event to JetStream.
func publish(ctx context.Context, js jetstream.JetStream, subject string, event EventAPIAuditLogged) error {
	if js == nil {
		return fmt.Errorf("nil jetstream client")
	}
	if event.MsgID() == "" {
		return fmt.Errorf("empty message id")
	}

	payload, err := marshalEvent(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	if _, err := js.Publish(ctx, subject, payload, jetstream.WithMsgID(event.MsgID())); err != nil {
		return fmt.Errorf("publish audit event: %w", err)
	}

	return nil
}
