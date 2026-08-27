package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mhmdkzr/app/internal/app"
	"github.com/mhmdkzr/app/internal/config"
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
	a := app.App{
		Cfg: config.Config{Server: config.ServerConfig{BasePath: "/api"}},
		Mux: http.NewServeMux(),
	}
	RegisterRoutes(a, Route{
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
	a := app.App{
		Cfg: config.Config{Server: config.ServerConfig{BasePath: "/api"}},
		Mux: http.NewServeMux(),
	}
	Handle(a, http.MethodGet, "/health", func(w http.ResponseWriter, _ *http.Request) {
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
