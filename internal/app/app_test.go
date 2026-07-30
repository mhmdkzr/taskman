package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mhmdkzr/app/internal/config"
	"github.com/mhmdkzr/app/pkg/resp"
)

func TestJoinBasePath(t *testing.T) {
	cases := []struct {
		name     string
		basePath string
		route    string
		want     string
	}{
		{name: "root_with_slash", basePath: "/", route: "/assets", want: "/assets"},
		{name: "root_without_slash", basePath: "/", route: "assets", want: "/assets"},
		{name: "clean_base_path", basePath: "/api/", route: "", want: "/api"},
		{name: "joined_path", basePath: "/api", route: "/assets/{asset_id}", want: "/api/assets/{asset_id}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := joinBasePath(tc.basePath, tc.route); got != tc.want {
				t.Fatalf("joinBasePath: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	var called bool
	a := App{
		Cfg: config.Config{Server: config.ServerConfig{BasePath: "/api"}},
		Mux: http.NewServeMux(),
	}
	a.RegisterRoutes(Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	a.Mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got=%d want=%d", rr.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("expected registered handler to be called")
	}
}

func TestHandle(t *testing.T) {
	var called bool
	a := App{
		Cfg: config.Config{Server: config.ServerConfig{BasePath: "/api"}},
		Mux: http.NewServeMux(),
	}
	a.Handle(http.MethodGet, "/health", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	a.Mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got=%d want=%d", rr.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("expected registered handler to be called")
	}
}

func TestWriteJSONAndError(t *testing.T) {
	rr := httptest.NewRecorder()
	resp.WriteJSON(rr, http.StatusCreated, map[string]string{"status": "ok"})

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
	resp.WriteHTTPError(rr, http.StatusBadRequest, errors.New("boom"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got=%d want=%d", rr.Code, http.StatusBadRequest)
	}
	if body := rr.Body.String(); !strings.Contains(body, `"error":"boom"`) {
		t.Fatalf("unexpected error body: %s", body)
	}
}

func TestWriteJSON_EncodeError(t *testing.T) {
	rr := httptest.NewRecorder()
	resp.WriteJSON(rr, http.StatusOK, map[string]chan int{"bad": make(chan int)})

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

func TestConfigLoad_FromEnv(t *testing.T) {
	t.Setenv("SKIP_ENV_AUTO_LOAD", "true")
	t.Setenv("NATS_URL", "nats://127.0.0.1:4222")
	t.Setenv("SERVER_BIND_ADDR", "127.0.0.1:9090")
	t.Setenv("SERVER_BASE_PATH", "/api")
	t.Setenv("SERVER_TIMEOUT", "5s")
	t.Setenv("SERVER_SHUTDOWN_TIMEOUT", "30s")
	t.Setenv("LOGGER_FORMAT", "json")
	t.Setenv("LOGGER_LEVEL", "warn")
	t.Setenv("POSTGRES_HOST", "127.0.0.1")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_DATABASE", "app")
	t.Setenv("POSTGRES_SSLMODE", "disable")
	t.Setenv("POSTGRES_AUTO_MIGRATE", "true")
	t.Setenv("TEMPORAL_HOST", "127.0.0.1:7233")
	t.Setenv("TEMPORAL_NAMESPACE", "default")
	t.Setenv("AUDIT_LOG_TIMEOUT", "10s")
	t.Setenv("NOTIFIER_ENABLED", "false")
	t.Setenv("NOTIFIER_TELEGRAM_BOT_TOKEN", "test-bot-token")
	t.Setenv("NOTIFIER_TELEGRAM_CHANNEL_ID", "12345")

	var cfg config.Config
	if err := cfg.Load(); err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Server.BindAddr != "127.0.0.1:9090" {
		t.Fatalf("server bind mismatch: %s", cfg.Server.BindAddr)
	}
	if cfg.Logger.Format != "json" {
		t.Fatalf("logger format mismatch: %s", cfg.Logger.Format)
	}
	if cfg.Database.Database != "app" {
		t.Fatalf("postgres db mismatch: %s", cfg.Database.Database)
	}
	if cfg.Database.User != "postgres" {
		t.Fatalf("postgres user mismatch: %s", cfg.Database.User)
	}
	if cfg.Temporal.Namespace != "default" {
		t.Fatalf("temporal namespace mismatch: %s", cfg.Temporal.Namespace)
	}
	if cfg.Database.AutoMigrate != true {
		t.Fatalf("auto_migrate mismatch: %v", cfg.Database.AutoMigrate)
	}
	if cfg.NATS.URL != "nats://127.0.0.1:4222" {
		t.Fatalf("nats url mismatch: %s", cfg.NATS.URL)
	}
	if cfg.Notifier.Enabled != false {
		t.Fatalf("notifier enabled mismatch: %v", cfg.Notifier.Enabled)
	}
	if cfg.Notifier.Telegram.BotToken != "test-bot-token" {
		t.Fatalf("notifier telegram bot token mismatch: %s", cfg.Notifier.Telegram.BotToken)
	}
	if cfg.Notifier.Telegram.ChannelID != 12345 {
		t.Fatalf("notifier telegram channel id mismatch: %d", cfg.Notifier.Telegram.ChannelID)
	}
}
