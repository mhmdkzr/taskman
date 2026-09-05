// Package browser provides the shared Client backing the browser_* agent tools.
//
// Each tool lives in its own subpackage (navigate, extract, click, fill,
// scroll, back, forward, reset) and takes the Client as a dependency. The
// Client owns a single lazily-launched Chromium instance and one current page.
// State is shared across all callers; a mutex serializes concurrent access.
//
// Shared state note: the browser and current page are currently shared across
// all agent sessions in the process. In the future this should be separated
// per user/session so parallel sessions do not interfere (e.g. via per-session
// incognito contexts or a page pool).
package browser

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"github.com/mhmdkzr/loop/internal/app/config"
)

// maxTextBytes caps the text returned by extract and other observation helpers.
const maxTextBytes = 64 << 10

// defaultTimeout is the per-operation timeout when none is configured.
const defaultTimeout = 60 * time.Second

// Client holds the configuration and lazily-initialized browser state shared
// by the browser_* tools.
type Client struct {
	cfg config.BrowserConfig

	mu       sync.Mutex
	browser  *rod.Browser
	page     *rod.Page
	launcher *launcher.Launcher
	cancel   context.CancelFunc
	initErr  error
	once     sync.Once
}

type PageResult struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

type ExtractResult struct {
	URL      string `json:"url"`
	Title    string `json:"title,omitempty"`
	Selector string `json:"selector,omitempty"`
	Text     string `json:"text"`
}

type ClickResult struct {
	Clicked string `json:"clicked"`
	URL     string `json:"url"`
	Title   string `json:"title,omitempty"`
}

type FillResult struct {
	Filled string `json:"filled"`
	Text   string `json:"text"`
}

type ScrollResult struct {
	Scrolled  bool   `json:"scrolled"`
	Direction string `json:"direction,omitempty"`
	Selector  string `json:"selector,omitempty"`
}

type ResetResult struct {
	Reset bool `json:"reset"`
}

// NewClient returns a Client for the given config.
func NewClient(cfg config.BrowserConfig) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	return &Client{cfg: cfg}
}

// NewClientFromConfig returns a Client from the given config.
func NewClientFromConfig(cfg config.BrowserConfig) *Client {
	return NewClient(cfg)
}

// Configured reports whether the browser tools are enabled.
func (c *Client) Configured() bool {
	return c != nil && c.cfg.Enabled
}

// effectiveTimeout returns the configured timeout or the default.
//
//nolint:funcorder // Internal lifecycle helpers are grouped before operations.
func (c *Client) effectiveTimeout() time.Duration {
	if c.cfg.Timeout > 0 {
		return c.cfg.Timeout
	}
	return defaultTimeout
}

// ensureBrowserLocked lazily launches or connects to a browser. Caller must
// hold c.mu.
//
//nolint:funcorder // Internal lifecycle helpers are grouped before operations.
func (c *Client) ensureBrowserLocked() error {
	c.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		c.cancel = cancel

		var b *rod.Browser
		var l *launcher.Launcher
		var controlURL string
		var err error

		if c.cfg.CDPURL != "" {
			b = rod.New().ControlURL(c.cfg.CDPURL).Context(ctx)
		} else {
			l = launcher.New()
			if c.cfg.Bin != "" {
				l = l.Bin(c.cfg.Bin)
			}
			l = l.Headless(c.cfg.Headless)
			controlURL, err = l.Launch()
			if err != nil {
				cancel()
				c.initErr = fmt.Errorf("browser: launch: %w", err)
				return
			}
			b = rod.New().ControlURL(controlURL).Context(ctx)
			c.launcher = l
		}

		b = b.Timeout(c.effectiveTimeout())

		if err := b.Connect(); err != nil {
			if l != nil {
				l.Kill()
			}
			cancel()
			c.initErr = fmt.Errorf("browser: connect: %w", err)
			return
		}

		c.browser = b
		slog.Info("browser launched", "headless", c.cfg.Headless, "control_url", controlURL)
	})
	return c.initErr
}

// ensurePageLocked returns the current page, creating one if needed. Caller
// must hold c.mu and have ensured the browser.
//
//nolint:funcorder // Internal lifecycle helpers are grouped before operations.
func (c *Client) ensurePageLocked() (*rod.Page, error) {
	if c.page != nil {
		return c.page, nil
	}
	if c.browser == nil {
		return nil, fmt.Errorf("browser: not initialized")
	}
	p, err := c.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("browser: create page: %w", err)
	}
	p = p.Timeout(c.effectiveTimeout())
	c.page = p
	return p, nil
}

// pageInfo returns the current URL and title for the page. It never fails the
// caller on title errors; it returns what it can.
func pageInfo(p *rod.Page) (string, string) {
	info, err := p.Info()
	pageURL := ""
	if err == nil && info != nil {
		pageURL = info.URL
	} else {
		// Fallback via JS.
		if obj, err2 := p.Eval("() => location.href"); err2 == nil && obj != nil {
			if v := obj.Value.String(); v != "" {
				pageURL = strings.Trim(v, `"`)
			}
		}
	}
	title := ""
	if obj, err := p.Eval("() => document.title"); err == nil && obj != nil {
		if v := obj.Value.String(); v != "" {
			title = strings.Trim(v, `"`)
		}
	}
	return pageURL, title
}

// truncate returns s capped to maxTextBytes with a notice when truncated.
func truncate(s string) string {
	if len(s) <= maxTextBytes {
		return s
	}
	return s[:maxTextBytes] + fmt.Sprintf("\n[output truncated after %d bytes]", maxTextBytes)
}

// validateURL validates that raw is a non-empty http or https URL.
func validateURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url %q must use http or https", raw)
	}
	if u.Host == "" {
		return fmt.Errorf("url %q must have a host", raw)
	}
	return nil
}

// validateSelector validates a non-empty CSS selector.
func validateSelector(sel string) error {
	if strings.TrimSpace(sel) == "" {
		return fmt.Errorf("selector is required")
	}
	return nil
}

// Navigate navigates the current page to rawURL and waits for load.
func (c *Client) Navigate(rawURL string) (PageResult, error) {
	if !c.Configured() {
		return PageResult{}, fmt.Errorf("browser_navigate: not configured: set BROWSER_ENABLED=true")
	}
	if err := validateURL(rawURL); err != nil {
		return PageResult{}, fmt.Errorf("browser_navigate: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureBrowserLocked(); err != nil {
		return PageResult{}, err
	}
	p, err := c.ensurePageLocked()
	if err != nil {
		return PageResult{}, err
	}

	if err := p.Navigate(rawURL); err != nil {
		return PageResult{}, fmt.Errorf("browser_navigate: navigate %q: %w", rawURL, err)
	}
	if err := p.WaitLoad(); err != nil {
		// WaitLoad can fail on navigation races; still return current state.
		slog.Warn("browser navigate wait load", "url", rawURL, "error", err)
	}

	finalURL, title := pageInfo(p)
	return PageResult{URL: finalURL, Title: title}, nil
}

// Extract returns visible text for the page or a specific element.
func (c *Client) Extract(selector string) (ExtractResult, error) {
	if !c.Configured() {
		return ExtractResult{}, fmt.Errorf("browser_extract: not configured: set BROWSER_ENABLED=true")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureBrowserLocked(); err != nil {
		return ExtractResult{}, err
	}
	p, err := c.ensurePageLocked()
	if err != nil {
		return ExtractResult{}, err
	}

	// No navigation yet — page may be blank.
	pageURL, title := pageInfo(p)

	var text string
	sel := strings.TrimSpace(selector)
	if sel == "" {
		el, err := p.Element("body")
		if err != nil {
			return ExtractResult{}, fmt.Errorf("browser_extract: find body: %w", err)
		}
		t, err := el.Text()
		if err != nil {
			return ExtractResult{}, fmt.Errorf("browser_extract: read body text: %w", err)
		}
		text = t
	} else {
		el, err := p.Element(sel)
		if err != nil {
			return ExtractResult{}, fmt.Errorf("browser_extract: find %q: %w", sel, err)
		}
		t, err := el.Text()
		if err != nil {
			return ExtractResult{}, fmt.Errorf("browser_extract: read %q text: %w", sel, err)
		}
		text = t
	}

	return ExtractResult{URL: pageURL, Title: title, Selector: sel, Text: truncate(strings.TrimSpace(text))}, nil
}

// Click clicks the element matching selector.
func (c *Client) Click(selector string) (ClickResult, error) {
	if !c.Configured() {
		return ClickResult{}, fmt.Errorf("browser_click: not configured: set BROWSER_ENABLED=true")
	}
	if err := validateSelector(selector); err != nil {
		return ClickResult{}, fmt.Errorf("browser_click: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureBrowserLocked(); err != nil {
		return ClickResult{}, err
	}
	p, err := c.ensurePageLocked()
	if err != nil {
		return ClickResult{}, err
	}

	el, err := p.Element(selector)
	if err != nil {
		return ClickResult{}, fmt.Errorf("browser_click: find %q: %w", selector, err)
	}
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return ClickResult{}, fmt.Errorf("browser_click: click %q: %w", selector, err)
	}
	// Give the page a moment to start navigation or DOM updates.
	if err := p.WaitLoad(); err != nil {
		slog.Warn("browser click wait for page load", "error", err)
	}

	pageURL, title := pageInfo(p)
	return ClickResult{Clicked: selector, URL: pageURL, Title: title}, nil
}

// Fill focuses the element matching selector and types text into it.
func (c *Client) Fill(selector, text string) (FillResult, error) {
	if !c.Configured() {
		return FillResult{}, fmt.Errorf("browser_fill: not configured: set BROWSER_ENABLED=true")
	}
	if err := validateSelector(selector); err != nil {
		return FillResult{}, fmt.Errorf("browser_fill: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureBrowserLocked(); err != nil {
		return FillResult{}, err
	}
	p, err := c.ensurePageLocked()
	if err != nil {
		return FillResult{}, err
	}

	el, err := p.Element(selector)
	if err != nil {
		return FillResult{}, fmt.Errorf("browser_fill: find %q: %w", selector, err)
	}
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return FillResult{}, fmt.Errorf("browser_fill: focus %q: %w", selector, err)
	}
	// Clear existing content via JS then input.
	clearInput := rod.Eval(`() => { this.value = ""; this.dispatchEvent(new Event("input", {bubbles:true})); }`)
	if _, err := el.Evaluate(clearInput); err != nil {
		slog.Warn("browser fill clear", "selector", selector, "error", err)
	}
	if err := el.Input(text); err != nil {
		return FillResult{}, fmt.Errorf("browser_fill: type into %q: %w", selector, err)
	}

	return FillResult{Filled: selector, Text: text}, nil
}

// Scroll scrolls the page. If selector is non-empty it scrolls the element
// into view; otherwise direction controls window scrolling.
func (c *Client) Scroll(direction, selector string) (ScrollResult, error) {
	if !c.Configured() {
		return ScrollResult{}, fmt.Errorf("browser_scroll: not configured: set BROWSER_ENABLED=true")
	}

	// Validate direction before launching the browser so bad inputs fail fast.
	sel := strings.TrimSpace(selector)
	if sel == "" {
		dir := strings.ToLower(strings.TrimSpace(direction))
		if dir != "" && dir != "up" && dir != "down" && dir != "top" && dir != "bottom" {
			return ScrollResult{}, fmt.Errorf(
				"browser_scroll: unknown direction %q: want up, down, top, or bottom",
				direction,
			)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureBrowserLocked(); err != nil {
		return ScrollResult{}, err
	}
	p, err := c.ensurePageLocked()
	if err != nil {
		return ScrollResult{}, err
	}

	if sel != "" {
		el, err := p.Element(sel)
		if err != nil {
			return ScrollResult{}, fmt.Errorf("browser_scroll: find %q: %w", sel, err)
		}
		if err := el.ScrollIntoView(); err != nil {
			return ScrollResult{}, fmt.Errorf("browser_scroll: scroll %q into view: %w", sel, err)
		}
		return ScrollResult{Scrolled: true, Selector: sel}, nil
	}

	dir := strings.ToLower(strings.TrimSpace(direction))
	if dir == "" {
		dir = "down"
	}
	var js string
	switch dir {
	case "up":
		js = "() => window.scrollBy(0, -Math.floor(window.innerHeight * 0.8))"
	case "down":
		js = "() => window.scrollBy(0, Math.floor(window.innerHeight * 0.8))"
	case "top":
		js = "() => window.scrollTo(0, 0)"
	case "bottom":
		js = "() => window.scrollTo(0, document.body.scrollHeight)"
	default:
		return ScrollResult{}, fmt.Errorf(
			"browser_scroll: unknown direction %q: want up, down, top, or bottom",
			direction,
		)
	}
	if _, err := p.Eval(js); err != nil {
		return ScrollResult{}, fmt.Errorf("browser_scroll: %w", err)
	}
	return ScrollResult{Scrolled: true, Direction: dir}, nil
}

// Back navigates one step back in history.
//
//nolint:dupl // Back and Forward intentionally share the same navigation lifecycle.
func (c *Client) Back() (PageResult, error) {
	if !c.Configured() {
		return PageResult{}, fmt.Errorf("browser_back: not configured: set BROWSER_ENABLED=true")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureBrowserLocked(); err != nil {
		return PageResult{}, err
	}
	p, err := c.ensurePageLocked()
	if err != nil {
		return PageResult{}, err
	}

	if err := p.NavigateBack(); err != nil {
		return PageResult{}, fmt.Errorf("browser_back: %w", err)
	}
	if err := p.WaitLoad(); err != nil {
		slog.Warn("browser back wait for page load", "error", err)
	}

	pageURL, title := pageInfo(p)
	return PageResult{URL: pageURL, Title: title}, nil
}

// Forward navigates one step forward in history.
//
//nolint:dupl // Back and Forward intentionally share the same navigation lifecycle.
func (c *Client) Forward() (PageResult, error) {
	if !c.Configured() {
		return PageResult{}, fmt.Errorf("browser_forward: not configured: set BROWSER_ENABLED=true")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureBrowserLocked(); err != nil {
		return PageResult{}, err
	}
	p, err := c.ensurePageLocked()
	if err != nil {
		return PageResult{}, err
	}

	if err := p.NavigateForward(); err != nil {
		return PageResult{}, fmt.Errorf("browser_forward: %w", err)
	}
	if err := p.WaitLoad(); err != nil {
		slog.Warn("browser forward wait for page load", "error", err)
	}

	pageURL, title := pageInfo(p)
	return PageResult{URL: pageURL, Title: title}, nil
}

// Reset closes the current page so the next operation starts fresh.
func (c *Client) Reset() (ResetResult, error) {
	if !c.Configured() {
		return ResetResult{}, fmt.Errorf("browser_reset: not configured: set BROWSER_ENABLED=true")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.page != nil {
		if err := c.page.Close(); err != nil {
			slog.Warn("browser reset close page", "error", err)
		}
		c.page = nil
	}
	return ResetResult{Reset: true}, nil
}

// Close releases the browser resources. It is safe to call multiple times.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.page != nil {
		if err := c.page.Close(); err != nil {
			slog.Warn("browser close page", "error", err)
		}
		c.page = nil
	}
	if c.browser != nil {
		if err := c.browser.Close(); err != nil {
			slog.Warn("browser close", "error", err)
		}
		c.browser = nil
	}
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	if c.launcher != nil {
		c.launcher.Cleanup()
		c.launcher = nil
	}
	return nil
}
