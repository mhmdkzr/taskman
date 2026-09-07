// Package opencode provides access to the OpenCode Zen API for the app's
// configured provider: usage status and limits for the OpenCode Go plan,
// reporting how much of each quota window is used and when it resets.
package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mhmdkzr/loop/internal/app/config"
)

// WindowStatus is the provider's status for one quota window.
type WindowStatus string

const (
	// StatusOK means the window still has quota left.
	StatusOK WindowStatus = "ok"
	// StatusRateLimited means the window's quota is exhausted until it resets.
	StatusRateLimited WindowStatus = "rate-limited"
)

// Usage is the OpenCode Go plan quota usage across its three windows.
type Usage struct {
	Rolling Window `json:"rolling"`
	Weekly  Window `json:"weekly"`
	Monthly Window `json:"monthly"`
}

// Window is the usage state of one quota window. Percent is the share of the
// window's quota already used (0-100); the share left is 100 - Percent.
type Window struct {
	Status   WindowStatus `json:"status"`
	Percent  int          `json:"percent"`
	ResetsAt time.Time    `json:"resets_at"`
}

var (
	// ErrNotConfigured is returned when the provider base URL or API key is
	// not configured.
	ErrNotConfigured = errors.New("opencode: provider is not configured")
	// ErrUnauthorized is returned when the provider rejects the configured
	// API key.
	ErrUnauthorized = errors.New("opencode: API key is missing or rejected")
	// ErrNoSubscription is returned when the workspace has no OpenCode Go
	// subscription.
	ErrNoSubscription = errors.New("opencode: OpenCode Go subscription required")
)

// usageResponse mirrors the upstream wire format of GET /usage
// (anomalyco/opencode PR #16513).
type usageResponse struct {
	Usage struct {
		Rolling windowResponse `json:"rolling"`
		Weekly  windowResponse `json:"weekly"`
		Monthly windowResponse `json:"monthly"`
	} `json:"usage"`
}

type windowResponse struct {
	Status   string `json:"status"`
	Percent  int    `json:"percent"`
	ResetsAt string `json:"resetsAt"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// FetchUsage queries the provider's OpenCode Go usage endpoint, which reports
// the rolling (~5h), weekly, and monthly quota windows as percentages of
// their limits with reset timestamps. Raw spend and dollar limits are not
// part of the endpoint's response.
func FetchUsage(ctx context.Context, cfg config.ProviderConfig) (Usage, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return Usage{}, fmt.Errorf("%w: base URL is empty", ErrNotConfigured)
	}
	if strings.TrimSpace(cfg.APIKeyOpenCode) == "" {
		return Usage{}, fmt.Errorf("%w: API key is empty", ErrNotConfigured)
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, strings.TrimRight(cfg.BaseURL, "/")+"/usage", nil)
	if err != nil {
		return Usage{}, fmt.Errorf("opencode: create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+cfg.APIKeyOpenCode)
	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return Usage{}, fmt.Errorf("opencode: request usage: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			slog.Error("close opencode usage response", "error", closeErr)
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Usage{}, fmt.Errorf("opencode: read usage response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Usage{}, upstreamError(response.StatusCode, body)
	}

	var payload usageResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return Usage{}, fmt.Errorf("opencode: decode usage: %w", err)
	}
	rolling, err := payload.Usage.Rolling.window()
	if err != nil {
		return Usage{}, err
	}
	weekly, err := payload.Usage.Weekly.window()
	if err != nil {
		return Usage{}, err
	}
	monthly, err := payload.Usage.Monthly.window()
	if err != nil {
		return Usage{}, err
	}
	return Usage{Rolling: rolling, Weekly: weekly, Monthly: monthly}, nil
}

// upstreamError converts a non-successful upstream response into a sentinel
// error when the status is known, wrapping the provider's error message.
func upstreamError(status int, body []byte) error {
	message := ""
	var payload errorResponse
	if json.Unmarshal(body, &payload) == nil {
		message = payload.Error.Message
	}

	var sentinel error
	switch status {
	case http.StatusUnauthorized:
		sentinel = ErrUnauthorized
	case http.StatusForbidden:
		sentinel = ErrNoSubscription
	}
	if sentinel == nil {
		if message != "" {
			return fmt.Errorf("opencode: usage returned HTTP %d: %s", status, message)
		}
		return fmt.Errorf("opencode: usage returned HTTP %d", status)
	}
	if message != "" {
		return fmt.Errorf("%w: %s", sentinel, message)
	}
	return sentinel
}

// window converts one upstream window into the domain type.
func (w windowResponse) window() (Window, error) {
	resetsAt, err := time.Parse(time.RFC3339, w.ResetsAt)
	if err != nil {
		return Window{}, fmt.Errorf("opencode: parse resets_at %q: %w", w.ResetsAt, err)
	}
	return Window{Status: WindowStatus(w.Status), Percent: w.Percent, ResetsAt: resetsAt}, nil
}
