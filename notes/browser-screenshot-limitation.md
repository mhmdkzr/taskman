# Browser Screenshot Limitation — goai Tool Results Are Text-Only

## Summary

`go-rod` can capture screenshots as `[]byte` PNG (`Page.Screenshot()`), but `goai` cannot return them as images from a tool. Tool outputs are `string`-only and are sent to the model as `provider.PartToolResult{ToolOutput string}` — the model never sees them as vision input.

## How goai Sends Images (provider contract)

Images are sent as **user/assistant message parts**, not tool results:

* `provider/types.go:117` — `PartType = "image"` (`PartImage`)
* `provider/types.go:398` — `type Part struct { Type PartType; URL string; MediaType string; Detail string; RemoteRef *RemoteFileRef }`
* `provider/types.go:403` — `URL` for images expects `data:image/png;base64,...` format (base64 **string**, not `[]byte` directly)
* Wire serialization `internal/openaicompat/messages.go:72,102` → `{"type":"image_url","image_url":{"url":..., "detail":...}}`
* Construction: `provider.Message{Role: provider.RoleUser, Content: []provider.Part{{Type: provider.PartImage, URL: "data:image/png;base64,"+base64.StdEncoding.EncodeToString(png), MediaType: "image/png", Detail: "high"}}}`

Alternative path for large files: `provider/types.go:14` `FileUpload{Reader, MediaType}` + `provider/types.go:44` `FileUploader.UploadFile` → `RemoteFileRef` → `Part{Type: PartImage, RemoteRef: ref}` (`provider/types.go:430`).

## Why Tools Can't Return Images

* `vendor/github.com/zendev-sh/goai/types.go` — `type Tool struct { Execute func(ctx context.Context, input json.RawMessage) (string, error) }`
* `vendor/github.com/zendev-sh/goai/generate.go:executeToolsParallel` collects `toolOutput{result string, err error}` and `provider/types.go:411` builds `Part{Type: PartToolResult, ToolOutput: result}`
* `internal/openaicompat/messages.go:44` serializes strictly as `{"role":"tool","tool_call_id":..., "content": ToolOutput}` — no `PartImage` branch.

Returning `base64` string from a tool would be treated as plain text; most vision models will not interpret it as an image.

## Consequence for `internal/agent/tools/browser`

* Current implementation `internal/agent/tools/browser/README.md` is **text-only** (`browser_extract` capped at 64 KiB) `internal/agent/tools/browser/browser.go:17`. Correct choice — screenshots would be invisible to the model.
* Adding `browser/screenshot` (`Page.Screenshot()` → `[]byte` → base64 `data:` URL) as a plain `goai.Tool` is insufficient.

## Workarounds (if screenshots are needed)

1. **No screenshot (recommended v1)** — keep `browser_extract` as observation channel. Simplest, no session-loop changes.

2. **Screenshot-as-follow-up user message** — keep `browser/screenshot` tool returning `data:image/png;base64,...` string, but extend `internal/agent/sessions/session.go:69` `Run()` to detect that prefix in `OnAfterToolExecute`/`OnBeforeStep` (`options.go:116`) and inject a new `provider.Message{Role:RoleUser, Content:[]Part{PartImage}}` before the next `GenerateText` step. Requires plumbing outside `tools/browser` (session loop, `AgentState` `options.go:150`, `WithMessages` `options.go:184`). Must also handle token cost (PNG ~100-500 KiB base64) and provider limits.

3. **Provider-defined computer tool** — use `ToolDefinition.ProviderDefinedType = "computer_20250124"` (`provider/types.go:450`) where the provider natively handles screenshots; bypasses custom `rod` code.

4. **Wait for upstream multimodal tool results** — would require `goai` to allow `Execute` to return `[]Part` or `ToolOutput` with `PartImage`, or to accept `RemoteFileRef`.

## References

* `provider/types.go:117,403,430` — `PartImage` contract
* `types.go:Tool.Execute` — text-only return
* `internal/openaicompat/messages.go:44,72,102` — tool vs image serialization
* `internal/agent/tools/browser/browser.go:8` — shared-state comment; `browser.go:8` notes future per-session separation
* `internal/agent/sessions/session.go:69` — `WithTools`/`WithMessages` tool loop injection point

## Decision

Documented as limitation; do not add `browser_screenshot` until (2) or (4) is implemented. If requested, prefer (2) with explicit `RUN_E2E_TESTS` gated tests (PNG needs headless Chromium).
