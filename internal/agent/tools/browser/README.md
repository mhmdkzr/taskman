# Browser Tools

Agent tools for headless browser automation backed by [go-rod/rod](https://github.com/go-rod/rod) (Chrome DevTools Protocol).

## Overview

The `browser` package owns a single lazily-launched Chromium instance and one
current page, shared across all callers. Each action is a separate goai tool in
its own subpackage, mirroring the `files/` and `telegram/` patterns. All tools
take a `*browser.Client` dependency so the browser state is shared.

> **Shared state note:** the browser and current page are currently shared
> across all agent sessions in the process (serialized by a mutex). In the
> future this should be separated per user/session — e.g. via per-session
> incognito contexts or a page pool — so parallel sessions do not interfere.

Tool results are text-only (goai's `Tool.Execute` returns `string`); there is
no image/screenshot path.

## Configuration

`config.BrowserConfig` (`envPrefix: BROWSER_`):

| Env | Default | Description |
|-----|---------|-------------|
| `BROWSER_ENABLED` | `true` | Set `false` to disable all browser_* tools |
| `BROWSER_HEADLESS` | `true` | Headless mode |
| `BROWSER_BIN` | `""` | Override path to Chrome/Chromium binary |
| `BROWSER_CDP_URL` | `""` | Remote CDP endpoint; when set, no local browser is launched |
| `BROWSER_TIMEOUT` | `60s` | Per-operation timeout |

`launcher.New()` is used locally; `--no-sandbox` is added automatically inside
containers. The browser is bound to a long-lived context so it survives the
per-call goai `Execute` context.

## Tools

| Tool | Package | Input | Description |
|------|---------|-------|-------------|
| `browser_navigate` | `browser/navigate` | `url` | Navigate to a URL and wait for load |
| `browser_extract` | `browser/extract` | `selector?` | Visible text of the page or an element (byte-capped at 64 KiB) |
| `browser_click` | `browser/click` | `selector` | Click an element |
| `browser_fill` | `browser/fill` | `selector`, `text` | Focus, clear, and type into a field |
| `browser_scroll` | `browser/scroll` | `direction?`, `selector?` | Scroll the window (`up`/`down`/`top`/`bottom`) or an element into view |
| `browser_back` | `browser/back` | — | History back |
| `browser_forward` | `browser/forward` | — | History forward |
| `browser_reset` | `browser/reset` | — | Close the current page; next navigate creates a fresh one |

Construct with a shared client:

```go
client := browser.NewClientFromConfig(cfg.Browser)
tools := []goai.Tool{
    navigate.Tool(client),
    extract.Tool(client),
    click.Tool(client),
    fill.Tool(client),
    scroll.Tool(client),
    back.Tool(client),
    forward.Tool(client),
    reset.Tool(client),
}
```

## Typical Flow

```
browser_navigate {url: "https://example.com"}
browser_extract  {}
browser_click    {selector: "a.more"}
browser_extract  {selector: "article"}
browser_fill     {selector: "input[name=q]", text: "hello"}
browser_scroll   {direction: "down"}
browser_reset    {}
```

## Testing

- Pure unit tests (URL/selector validation) run under `go test ./...`.
- Browser-backed integration tests (httptest server + headless Chromium) are gated behind `RUN_E2E_TESTS=1` via `testenv.SkipIfE2ETestsDisabled` and live in `*_integration_test.go`.
