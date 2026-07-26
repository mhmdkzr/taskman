package logging

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mhmdkzr/app/pkg/middleware"
)

func TestLogging_InfoLevelFor2xx(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := middleware.NewResponseRecorder(httptest.NewRecorder())
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r = middleware.WithRecorder(r, rec)

	New()(handler).ServeHTTP(rec, r)

	if buf.Len() == 0 {
		t.Fatal("expected log output")
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"level":"INFO"`)) {
		t.Fatalf("expected INFO level, got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"method":"GET"`)) {
		t.Fatalf("expected method attr, got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"path":"/test"`)) {
		t.Fatalf("expected path attr, got: %s", buf.String())
	}
}

func TestLogging_ErrorLevelFor5xx(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	rec := middleware.NewResponseRecorder(httptest.NewRecorder())
	r := httptest.NewRequest(http.MethodGet, "/error", nil)
	r = middleware.WithRecorder(r, rec)

	New()(handler).ServeHTTP(rec, r)

	if !bytes.Contains(buf.Bytes(), []byte(`"level":"ERROR"`)) {
		t.Fatalf("expected ERROR level for 500, got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"status":500`)) {
		t.Fatalf("expected status attr, got: %s", buf.String())
	}
}

func TestLogging_WarnLevelFor4xx(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	rec := middleware.NewResponseRecorder(httptest.NewRecorder())
	r := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	r = middleware.WithRecorder(r, rec)

	New()(handler).ServeHTTP(rec, r)

	if !bytes.Contains(buf.Bytes(), []byte(`"level":"WARN"`)) {
		t.Fatalf("expected WARN level for 404, got: %s", buf.String())
	}
}

func TestLogging_IncludesContextData(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := middleware.NewResponseRecorder(httptest.NewRecorder())
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := middleware.WithClientIP(r.Context(), "10.0.0.1")
	ctx = middleware.WithRequestID(ctx, "req-1")
	r = r.WithContext(ctx)
	r = middleware.WithRecorder(r, rec)

	New()(handler).ServeHTTP(rec, r)

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte(`"ip":"10.0.0.1"`)) {
		t.Fatalf("expected ip attr, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"request_id":"req-1"`)) {
		t.Fatalf("expected request_id attr, got: %s", output)
	}
}

func TestLogging_IncludesDuration(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := middleware.NewResponseRecorder(httptest.NewRecorder())
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = middleware.WithRecorder(r, rec)

	New()(handler).ServeHTTP(rec, r)

	if !bytes.Contains(buf.Bytes(), []byte(`"duration"`)) {
		t.Fatalf("expected duration attr, got: %s", buf.String())
	}
}

func TestLogging_WithoutRecorderStillLogs(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	New()(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if buf.Len() == 0 {
		t.Fatal("expected log output even without recorder")
	}
}
