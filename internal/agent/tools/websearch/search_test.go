package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zendev-sh/goai"

	"github.com/mhmdkzr/loop/internal/app/config"
)

func execTool(t *testing.T, tool goai.Tool, raw string) (string, error) {
	t.Helper()
	return tool.Execute(context.Background(), json.RawMessage(raw))
}

func TestNewClientFromConfig(t *testing.T) {
	if c := NewClientFromConfig(config.TavilyConfig{APIKey: "k"}); !c.Configured() {
		t.Error("not configured after NewClientFromConfig")
	}
	if c := NewClientFromConfig(config.TavilyConfig{}); c.Configured() {
		t.Error("should be disabled with empty config")
	}
	if c := NewClient(""); c.Configured() {
		t.Error("should be disabled with empty api key")
	}
}

func TestToolFormatsResults(t *testing.T) {
	c := NewClient("token-123")
	c.search = func(ctx context.Context, baseURL string, client *http.Client, apiKey, query string, maxResults int) ([]SearchResult, error) {
		if apiKey != "token-123" {
			t.Errorf("api key = %q, want token-123", apiKey)
		}
		if query != "golang generics" {
			t.Errorf("query = %q, want golang generics", query)
		}
		if maxResults != 5 {
			t.Errorf("maxResults = %d, want default 5", maxResults)
		}
		return []SearchResult{
			{Title: "Go by Example", URL: "https://gobyexample.com", Content: "Learn Go"},
			{Title: "Go Docs", URL: "https://go.dev", Content: ""},
		}, nil
	}

	out, err := execTool(t, Tool(c), `{"query":" golang generics "}`)
	if err != nil {
		t.Fatal(err)
	}
	var got Output
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 2 || got.Results[0].Title != "Go by Example" || got.Results[1].URL != "https://go.dev" {
		t.Errorf("results = %+v", got.Results)
	}
}

func TestToolMaxResults(t *testing.T) {
	c := NewClient("k")
	c.search = func(ctx context.Context, _ string, _ *http.Client, _ string, _ string, maxResults int) ([]SearchResult, error) {
		if maxResults != 3 {
			t.Errorf("maxResults = %d, want 3", maxResults)
		}
		return nil, nil
	}
	out, err := execTool(t, Tool(c), `{"query":"x","max_results":3}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"results":null}` {
		t.Errorf("out = %q, want empty results", out)
	}
}

func TestToolNoResults(t *testing.T) {
	c := NewClient("k")
	c.search = func(context.Context, string, *http.Client, string, string, int) ([]SearchResult, error) {
		return nil, nil
	}
	out, err := execTool(t, Tool(c), `{"query":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != `{"results":null}` {
		t.Errorf("out = %q, want empty results", out)
	}
}

func TestToolErrors(t *testing.T) {
	c := NewClient("")

	if _, err := execTool(t, Tool(c), `{`); err == nil {
		t.Error("expected invalid json error")
	}
	if _, err := execTool(t, Tool(c), `{"query":""}`); err == nil {
		t.Error("expected empty query error")
	}
	if _, err := execTool(t, Tool(c), `{"query":"x"}`); err == nil {
		t.Error("expected not-configured error")
	}

	c = NewClient("k")
	c.search = func(context.Context, string, *http.Client, string, string, int) ([]SearchResult, error) {
		return nil, nil
	}
	for _, args := range []string{`{"query":"x","max_results":0}`, `{"query":"x","max_results":11}`} {
		if _, err := execTool(t, Tool(c), args); err == nil {
			t.Errorf("expected max_results error for %s", args)
		}
	}
}

func TestToolSearchError(t *testing.T) {
	c := NewClient("k")
	c.search = func(context.Context, string, *http.Client, string, string, int) ([]SearchResult, error) {
		return nil, context.DeadlineExceeded
	}
	if _, err := execTool(t, Tool(c), `{"query":"x"}`); err == nil {
		t.Error("expected search error")
	}
}

func TestSearchTavily(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[{"title":"T","url":"https://x","content":"c"}]}`)
	}))
	defer srv.Close()

	results, err := searchTavily(context.Background(), srv.URL, &http.Client{}, "tok", "query here", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "T" || results[0].URL != "https://x" {
		t.Fatalf("results = %+v", results)
	}
	if gotPath != "/search" {
		t.Errorf("path = %q, want /search", gotPath)
	}
	if gotBody["api_key"] != "tok" || gotBody["query"] != "query here" || gotBody["max_results"] != float64(3) {
		t.Errorf("body = %+v", gotBody)
	}
}

func TestSearchTavilyHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "bad key")
	}))
	defer srv.Close()

	_, err := searchTavily(context.Background(), srv.URL, &http.Client{}, "tok", "q", 3)
	if err == nil || !strings.Contains(err.Error(), "search status 401") {
		t.Fatalf("err = %v, want status 401 error", err)
	}
}

func TestSearchTavilyDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":`)
	}))
	defer srv.Close()

	if _, err := searchTavily(context.Background(), srv.URL, &http.Client{}, "tok", "q", 3); err == nil {
		t.Error("expected decode error")
	}
}

func TestSearchTavilyNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	if _, err := searchTavily(context.Background(), url, &http.Client{}, "tok", "q", 3); err == nil {
		t.Error("expected network error")
	}
}
