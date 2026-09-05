package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mhmdkzr/loop/internal/app"
	"github.com/mhmdkzr/loop/internal/app/config"
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
		{name: "slashless_base_path", basePath: "api", route: "/assets", want: "api/assets"},
		{name: "slashless_base_path_with_slashless_route", basePath: "api", route: "assets", want: "api/assets"},
		{name: "slashless_base_path_empty_route", basePath: "api", route: "", want: "api"},
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

func TestHandle_RejectsEmptyMethod(t *testing.T) {
	a := app.App{
		Cfg: config.Config{Server: config.ServerConfig{BasePath: "/api"}},
		Mux: http.NewServeMux(),
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Handle to panic on an empty method")
		}
	}()
	Handle(a, "", "/health", func(w http.ResponseWriter, _ *http.Request) {})
}

func TestRegisterRoutes_MultipleRoutes(t *testing.T) {
	a := app.App{
		Cfg: config.Config{Server: config.ServerConfig{BasePath: "/api"}},
		Mux: http.NewServeMux(),
	}
	RegisterRoutes(a,
		Route{
			Method: http.MethodGet,
			Path:   "/health",
			Handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
		},
		Route{
			Method: http.MethodPost,
			Path:   "/things",
			Handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
			},
		},
	)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "first_route", method: http.MethodGet, path: "/api/health", want: http.StatusNoContent},
		{name: "second_route", method: http.MethodPost, path: "/api/things", want: http.StatusCreated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			a.Mux.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status mismatch: got=%d want=%d", rr.Code, tc.want)
			}
		})
	}
}

func TestRegisterRoutes_EmptyBasePathDefaultsToRoot(t *testing.T) {
	for _, basePath := range []string{"", "   "} {
		t.Run("base_path_"+basePath, func(t *testing.T) {
			a := app.App{
				Cfg: config.Config{Server: config.ServerConfig{BasePath: basePath}},
				Mux: http.NewServeMux(),
			}
			RegisterRoutes(a, Route{
				Method: http.MethodGet,
				Path:   "/health",
				Handler: func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
			})

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rr := httptest.NewRecorder()
			a.Mux.ServeHTTP(rr, req)
			if rr.Code != http.StatusNoContent {
				t.Fatalf("route not reachable at unprefixed path: got=%d want=%d", rr.Code, http.StatusNoContent)
			}
		})
	}
}

func TestRegisterRoutes_NegativePaths(t *testing.T) {
	a := app.App{
		Cfg: config.Config{Server: config.ServerConfig{BasePath: "/api"}},
		Mux: http.NewServeMux(),
	}
	RegisterRoutes(a, Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	})

	for _, tc := range []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "wrong_method", method: http.MethodPost, path: "/api/health", want: http.StatusMethodNotAllowed},
		{name: "unknown_path", method: http.MethodGet, path: "/api/nope", want: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			a.Mux.ServeHTTP(rr, req)
			if rr.Code != tc.want {
				t.Fatalf("status mismatch: got=%d want=%d", rr.Code, tc.want)
			}
		})
	}
}
