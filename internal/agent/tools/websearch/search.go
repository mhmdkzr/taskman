// Package websearch provides the websearch tool backed by the Tavily API.
// Build a Client with the API key, then pass it to Tool.
package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mhmdkzr/loop/internal/app/config"
)

const (
	defaultTavilyBaseURL = "https://api.tavily.com"
	defaultMaxResults    = 5
	maxSearchResults     = 10
)

// SearchResult is a single Tavily search hit.
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type output struct {
	Results []SearchResult `json:"results"`
}

// Client holds the credentials and dependencies shared by the websearch tool.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	search  func(ctx context.Context, baseURL string, client *http.Client, apiKey, query string, maxResults int) ([]SearchResult, error)
}

// NewClient returns a Client for the given API key. An empty key disables the
// tool; its execution then fails until the Client has a key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultTavilyBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
		search:  searchTavily,
	}
}

// NewClientFromConfig returns a Client from the given config. An empty key
// disables the tool.
func NewClientFromConfig(cfg config.TavilyConfig) *Client {
	return NewClient(cfg.APIKey)
}

// Configured reports whether credentials for the tool are present.
func (c *Client) Configured() bool {
	return c != nil && c.apiKey != ""
}

func execute(ctx context.Context, c *Client, in input) (output, error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return output{}, fmt.Errorf("websearch: query is required")
	}
	if !c.Configured() {
		return output{}, fmt.Errorf("websearch: not configured: set TAVILY_API_KEY")
	}

	maxResults := defaultMaxResults
	if in.MaxResults != nil {
		maxResults = *in.MaxResults
	}
	if maxResults < 1 || maxResults > maxSearchResults {
		return output{}, fmt.Errorf("websearch: max_results must be between 1 and %d", maxSearchResults)
	}

	results, err := c.search(ctx, c.baseURL, c.http, c.apiKey, query, maxResults)
	if err != nil {
		return output{}, fmt.Errorf("websearch: %w", err)
	}

	return output{Results: results}, nil
}

// searchTavily calls the Tavily /search endpoint and returns its results.
func searchTavily(
	ctx context.Context,
	baseURL string,
	client *http.Client,
	apiKey, query string,
	maxResults int,
) ([]SearchResult, error) {
	payload, err := json.Marshal(map[string]any{
		"api_key":     apiKey,
		"query":       query,
		"max_results": maxResults,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/search", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Tavily search: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("close Tavily search response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Tavily search response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payloadResp struct {
		Results []SearchResult `json:"results"`
	}
	if err := json.Unmarshal(body, &payloadResp); err != nil {
		return nil, err
	}
	return payloadResp.Results, nil
}
