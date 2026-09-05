# OpenCode Go Models

Fetched from `GET https://opencode.ai/zen/go/v1/models` on 2026-09-05 using
the configured OpenCode credential. The credential value was not written to
this file.

The endpoint returned 35 models. All models are owned by `opencode`.

## Models

| Model ID | Thinking options |
|---|---|
| `minimax-m3` | `thinking: {type: adaptive}` or `thinking: {type: disabled}` |
| `minimax-m2.7` | None exposed by OpenCode |
| `minimax-m2.5` | None exposed by OpenCode |
| `kimi-k3` | None exposed by OpenCode |
| `kimi-k2.7-code` | None exposed by OpenCode |
| `kimi-k2.6` | None exposed by OpenCode |
| `longcat-2.0` | `reasoning_effort`: `low`, `medium`, `high` |
| `kimi-k2.5` | None exposed by OpenCode |
| `glm-5.2` | `reasoning_effort`: `high`, `max` |
| `glm-5.3-flash` | None exposed by OpenCode |
| `glm-5.3` | None exposed by OpenCode |
| `glm-5.1` | None exposed by OpenCode |
| `glm-5` | None exposed by OpenCode |
| `deepseek-v4-pro` | `reasoning_effort`: `low`, `medium`, `high`, `max` |
| `deepseek-v4-flash` | `reasoning_effort`: `low`, `medium`, `high`, `max` |
| `deepseek-v4-flash-vision-exp` | `reasoning_effort`: `low`, `medium`, `high`, `max` |
| `qwen3.7-max` | None exposed by OpenCode |
| `qwen3.8-max` | None exposed by OpenCode |
| `qwen3.8-flash` | None exposed by OpenCode |
| `qwen3.7-plus` | None exposed by OpenCode |
| `qwen3.6-plus` | None exposed by OpenCode |
| `qwen3.5-plus` | None exposed by OpenCode |
| `mimo-v2-pro` | `reasoning_effort`: `low`, `medium`, `high` |
| `mimo-v2-omni` | `reasoning_effort`: `low`, `medium`, `high` |
| `mimo-v2.5-pro` | `reasoning_effort`: `low`, `medium`, `high` |
| `mimo-v2.5` | `reasoning_effort`: `low`, `medium`, `high` |
| `hy4-preview` | `reasoning_effort`: `low`, `medium`, `high` |
| `hy3` | `reasoning_effort`: `low`, `medium`, `high` |
| `hy3-preview` | `reasoning_effort`: `low`, `medium`, `high` |
| `gpt-5.6-luna` | `reasoning_effort`: `none`, `low`, `medium`, `high`, `xhigh` |
| `grok-4.5` | `reasoning_effort`: `low`, `medium`, `high` |
| `grok-4.6` | `reasoning_effort`: `low`, `medium`, `high` |
| `muse-spark-1.3-contributor` | `reasoning_effort`: `low`, `medium`, `high` |
| `muse-spark-1.2-contributor` | `reasoning_effort`: `low`, `medium`, `high` |
| `omen-alpha` | `reasoning_effort`: `low`, `medium`, `high` |

## Request Shape

For the OpenAI-compatible models, GoAI sends the selected option as a top-level
provider option:

```json
{
  "reasoning_effort": "high"
}
```

The `minimax-m3` variants use the model-specific `thinking` object instead of
`reasoning_effort`.

These options come from OpenCode's provider transform implementation:
<https://github.com/anomalyco/opencode/blob/dev/packages/opencode/src/provider/transform.ts>.
The live model catalog is dynamic and should be fetched again before treating
this file as authoritative.
