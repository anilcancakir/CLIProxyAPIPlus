# CLIProxyPlus Thinking Pipeline — Complete Analysis

> **Purpose**: LLM-friendly reference document for understanding how CLIProxyPlus handles extended
> thinking across all Claude models, including Anthropic's adaptive thinking (`type: "adaptive"`),
> suffix-based thinking control, cross-provider conversion (Claude ↔ OpenAI), and the full
> request/response transformation pipeline.
>
> **Sources**: CLIProxyPlus Go implementation (`internal/thinking/`, `internal/registry/`,
> `internal/translator/`), Anthropic official API documentation.
>
> **Date**: 2026-03-06

---

## 1. Anthropic API — Extended Thinking Specification

Anthropic's Messages API supports three thinking modes via the `thinking` parameter:

| Type | Structure | Status | Behavior |
|:-----|:----------|:-------|:---------|
| `adaptive` | `{"type": "adaptive"}` + optional `output_config.effort` | Recommended (4.6) | Model decides when and how much to think |
| `enabled` | `{"type": "enabled", "budget_tokens": N}` | Deprecated (4.6) | Forces thinking up to budget limit |
| `disabled` | `{"type": "disabled"}` or omit parameter | — | No extended reasoning |

### Adaptive Thinking (`type: "adaptive"`)

Introduced with Claude 4.6. Unlike `enabled`, adaptive thinking:

- Makes thinking optional — Claude evaluates request complexity and may skip thinking for trivial tasks
- Enables interleaved thinking between tool calls automatically
- Uses `output_config.effort` as a soft control instead of a hard token budget

### Effort Levels (`output_config.effort`)

| Level | Availability | Behavior |
|:------|:-------------|:---------|
| `max` | Opus 4.6 only | Deepest reasoning, no constraints |
| `high` | All Claude 4 models | Default behavior, high capability |
| `medium` | All Claude 4 models | Balanced speed/cost/performance |
| `low` | All Claude 4 models | Minimizes thinking, skips simple tasks |

### Request Example

```json
{
    "model": "claude-opus-4-6",
    "max_tokens": 16000,
    "thinking": {
        "type": "adaptive"
    },
    "output_config": {
        "effort": "medium"
    },
    "messages": [...]
}
```

### Response Format

Responses include `thinking` content blocks with a mandatory `signature` field:

```json
{
    "content": [
        {
            "type": "thinking",
            "thinking": "Let me analyze this step by step...",
            "signature": "WaUjzkypQ2mUEVM36O2TxuC06KN8xyfbJwyem2dw3URve..."
        },
        {
            "type": "text",
            "text": "The final answer is..."
        }
    ]
}
```

### Streaming Format

```
event: content_block_start
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "thinking", "thinking": ""}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": "Calculating..."}}

event: content_block_delta
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "signature_delta", "signature": "EqQBCgIYAhIM..."}}

event: content_block_stop
data: {"type": "content_block_stop", "index": 0}
```

### Key Constraints

- API version: `anthropic-version: 2023-06-01` (no beta headers required for adaptive)
- `budget_tokens` is **not used** with `type: "adaptive"` — it is only for `type: "enabled"`
- `budget_tokens` must be `< max_tokens`
- Tool use: only `tool_choice: "auto"` or `"none"` — forced tool choice returns 400
- Multi-turn continuity: previous assistant `thinking` blocks (including `signature`) must be passed
  back exactly as received

---

## 2. Model Registry — ThinkingSupport Configuration

Each Claude model has a `ThinkingSupport` struct that determines its thinking capabilities:

```go
type ThinkingSupport struct {
    Min            int      // Minimum allowed thinking budget (inclusive)
    Max            int      // Maximum allowed thinking budget (inclusive)
    ZeroAllowed    bool     // Whether 0 is valid (to disable thinking)
    DynamicAllowed bool     // Whether -1 is valid (dynamic thinking budget)
    Levels         []string // Discrete reasoning effort levels
}
```

Source: `internal/registry/model_registry.go`

### Claude Model Configurations

Source: `internal/registry/model_definitions_static_data.go`

| Model | ID | Min | Max | ZeroAllowed | Levels | Capability |
|:------|:---|:---:|:---:|:-----------:|:-------|:-----------|
| Claude 4.6 Opus | `claude-opus-4-6` | 1024 | 128000 | ✅ | `["low","medium","high","max"]` | Hybrid |
| Claude 4.6 Sonnet | `claude-sonnet-4-6` | 1024 | 128000 | ✅ | `["low","medium","high"]` | Hybrid |
| Claude 4.5 Sonnet | `claude-sonnet-4-5-20250929` | 1024 | 128000 | ✅ | — | Budget-only |
| Claude 4.5 Opus | `claude-opus-4-5-20251101` | 1024 | 128000 | ✅ | — | Budget-only |
| Claude 4.5 Haiku | `claude-haiku-4-5-20251001` | 1024 | 128000 | ✅ | — | Budget-only |
| Claude 4.1 Opus | `claude-opus-4-1-20250805` | 1024 | 128000 | ❌ | — | Budget-only |
| Claude 4 Opus | `claude-opus-4-20250514` | 1024 | 128000 | ❌ | — | Budget-only |
| Claude 4 Sonnet | `claude-sonnet-4-20250514` | 1024 | 128000 | ❌ | — | Budget-only |
| Claude 3.7 Sonnet | `claude-3-7-sonnet-20250219` | 1024 | 128000 | ❌ | — | Budget-only |

**Capability classification** (`internal/thinking/convert.go`):

| Capability | Condition | Models |
|:-----------|:----------|:-------|
| **Hybrid** | Has Min/Max AND Levels | claude-opus-4-6, claude-sonnet-4-6 |
| **Budget-only** | Has Min/Max, no Levels | claude-4.5, claude-4.1, claude-4, claude-3.7 |
| **Level-only** | Has Levels, no Min/Max | OpenAI models (gpt-5.x) |
| **None** | Thinking field is nil | claude-3.5 haiku, non-thinking models |

> [!NOTE]
> The `Levels` field is the key differentiator. Models with `Levels` support adaptive thinking
> (`type: "adaptive"` + `output_config.effort`). Models without `Levels` use manual thinking
> (`type: "enabled"` + `budget_tokens`).

---

## 3. Thinking Pipeline — Full Request Flow

The unified entry point is `ApplyThinking()` in `internal/thinking/apply.go`. Every request
passes through this pipeline regardless of source or target format.

### Pipeline Diagram

```
                    ┌─────────────────────────────────┐
                    │         CLIENT REQUEST           │
                    │  (Anthropic / OpenAI / Gemini)   │
                    └──────────────┬──────────────────┘
                                   │
                    ┌──────────────▼──────────────────┐
                    │   1. ParseSuffix(model)          │
                    │   "opus-4-6(high)" → "high"      │
                    └──────────────┬──────────────────┘
                                   │
              ┌────────────────────▼────────────────────┐
              │  2. extractThinkingConfig(body, format)  │
              │  OR parseSuffixToConfig (suffix wins)    │
              │         → ThinkingConfig                 │
              └────────────────────┬────────────────────┘
                                   │
              ┌────────────────────▼────────────────────┐
              │  3. ValidateConfig(config, modelInfo,    │
              │     fromFormat, toFormat, fromSuffix)     │
              │   - Auto-convert Mode ↔ Budget/Level     │
              │   - Clamp to model range                 │
              │   - Cross-family level clamping          │
              └────────────────────┬────────────────────┘
                                   │
                    ┌──────────────▼──────────────────┐
                    │  4. ProviderApplier.Apply()       │
                    └──┬───────────┬──────────┬───────┘
                       │           │          │
              ┌────────▼──┐ ┌─────▼────┐ ┌───▼──────┐
              │  Claude    │ │  OpenAI  │ │  Gemini  │
              │  Applier   │ │  Applier │ │  Applier │
              ├────────────┤ ├──────────┤ ├──────────┤
              │ adaptive:  │ │ level →  │ │ budget → │
              │ type+effort│ │ reason-  │ │ thinking │
              │ enabled:   │ │ ing_     │ │ Budget/  │
              │ type+budget│ │ effort   │ │ Level    │
              └────────────┘ └──────────┘ └──────────┘
```

---

## 4. Step 1 — Suffix Parsing

Source: `internal/thinking/suffix.go`

Users can append thinking suffixes to model names: `claude-opus-4-6(high)`.

### ParseSuffix

Extracts the base model name and raw suffix content:

```
"claude-opus-4-6(high)"   → ModelName="claude-opus-4-6", RawSuffix="high"
"claude-sonnet-4-5(8192)" → ModelName="claude-sonnet-4-5", RawSuffix="8192"
"claude-opus-4"           → ModelName="claude-opus-4", HasSuffix=false
```

### parseSuffixToConfig

Converts the raw suffix to a `ThinkingConfig` in priority order:

| Priority | Parser | Input Examples | Result |
|:--------:|:-------|:---------------|:-------|
| 1 | `ParseSpecialSuffix` | `(none)`, `(auto)`, `(-1)` | ModeNone / ModeAuto |
| 2 | `ParseLevelSuffix` | `(minimal)`, `(low)`, `(medium)`, `(high)`, `(xhigh)`, `(max)` | ModeLevel |
| 3 | `ParseNumericSuffix` | `(8192)`, `(0)`, `(128000)` | ModeBudget / ModeNone |

---

## 5. Step 2 — Config Extraction

Source: `internal/thinking/apply.go`

When no suffix is present, thinking config is extracted from the request body based on the
source format.

### extractClaudeConfig

Extracts from Anthropic Messages API format:

| Request Body | Extracted Config |
|:-------------|:-----------------|
| `thinking.type="disabled"` | `ModeNone, Budget=0` |
| `thinking.type="adaptive"` + `output_config.effort="high"` | `ModeLevel, Level="high"` |
| `thinking.type="adaptive"` (no effort) | Empty config (passthrough) |
| `thinking.type="enabled"` + `budget_tokens=16384` | `ModeBudget, Budget=16384` |
| `thinking.type="enabled"` (no budget) | `ModeAuto, Budget=-1` |

### extractOpenAIConfig

Extracts from OpenAI Chat Completions format:

| Request Body | Extracted Config |
|:-------------|:-----------------|
| `reasoning_effort="high"` | `ModeLevel, Level="high"` |
| `reasoning_effort="none"` | `ModeNone, Budget=0` |

> [!NOTE]
> Suffix config **always takes priority** over body config. This enables users to override
> thinking settings via the model name without modifying their request payload.

---

## 6. Step 3 — Validation

Source: `internal/thinking/validate.go`

`ValidateConfig` normalizes the thinking config against model capabilities and performs
auto-conversion between formats.

### Auto-Conversion Rules

| Config Mode | Model Capability | Conversion |
|:------------|:-----------------|:-----------|
| ModeLevel `(high)` | Budget-only | Level → Budget: `high` → 24576 |
| ModeLevel `(high)` | Hybrid | No conversion — level preserved |
| ModeBudget `(8192)` | Level-only | Budget → Level: 8192 → `medium` |
| ModeBudget `(8192)` | Budget-only | No conversion — budget preserved |
| ModeAuto `(auto)` | DynamicAllowed=true | No conversion |
| ModeAuto `(auto)` | DynamicAllowed=false, Level-only | → ModeLevel `medium` |
| ModeAuto `(auto)` | DynamicAllowed=false, Budget | → ModeBudget `(min+max)/2` |

### Budget Clamping

Budget values are clamped to the model's `[Min, Max]` range:

| Input | Model Range | ZeroAllowed | Result |
|:------|:------------|:-----------:|:-------|
| 512 | [1024, 128000] | — | Clamped to 1024 |
| 200000 | [1024, 128000] | — | Clamped to 128000 |
| 0 | [1024, 128000] | ✅ | 0 (thinking disabled) |
| 0 | [1024, 128000] | ❌ | Clamped to 1024 |

### Level Clamping

Unsupported levels are clamped to the nearest supported level:

```
Standard order: minimal → low → medium → high → xhigh → max

Model supports: ["low", "medium", "high"]
  "minimal" → clamp to "low" (nearest)
  "xhigh"   → clamp to "high" (nearest)
  "max"     → clamp to "high" (nearest)

Model supports: ["low", "medium", "high", "max"]
  "xhigh"   → clamp to "max" (nearest by index distance)
  "minimal" → clamp to "low" (nearest)
```

### Strict vs Cross-Family Validation

Two flags control validation strictness:

| Flag | Condition | Behavior |
|:-----|:----------|:---------|
| `strictBudget` | Same provider family + body config | Out-of-range budget → **error** |
| `strictBudget=false` | Cross-family or suffix | Out-of-range budget → **clamp** |
| `allowClampUnsupported` | Cross-family + target has levels | Unsupported level → **clamp** to nearest |
| `allowClampUnsupported=false` | Same provider family | Unsupported level → **error** |

Provider family groupings:

| Family | Members |
|:-------|:--------|
| Gemini | `gemini`, `gemini-cli`, `antigravity` |
| OpenAI | `openai`, `openai-response`, `codex` |
| Claude | `claude` (standalone) |

---

## 7. Step 4 — Provider Appliers

### Claude Applier

Source: `internal/thinking/provider/claude/apply.go`

| Config | Output (model has Levels) | Output (budget-only model) |
|:-------|:--------------------------|:---------------------------|
| ModeLevel, Level="high" | `thinking.type="adaptive"` + `output_config.effort="high"` | Fallback: level→budget 24576 + `thinking.type="enabled"` + `budget_tokens=24576` |
| ModeBudget, Budget=8192 | `thinking.type="enabled"` + `budget_tokens=8192` | `thinking.type="enabled"` + `budget_tokens=8192` |
| ModeAuto | `thinking.type="adaptive"` (no effort) | `thinking.type="enabled"` (no budget) |
| ModeNone | `thinking.type="disabled"` | `thinking.type="disabled"` |

Additional constraint: `budget_tokens` must be `< max_tokens`. The applier adjusts via
`normalizeClaudeBudget()`.

### OpenAI Applier

Source: `internal/thinking/provider/openai/apply.go`

| Config | Output |
|:-------|:-------|
| ModeLevel, Level="high" | `reasoning_effort="high"` |
| ModeNone (ZeroAllowed) | `reasoning_effort="none"` |
| ModeBudget (user-defined) | Budget→Level conversion then `reasoning_effort=X` |
| ModeAuto (user-defined) | `reasoning_effort="auto"` |

For user-defined models, `ModeBudget` triggers `ConvertBudgetToLevel()`:

| Budget | → Level |
|:-------|:--------|
| 0 | `none` |
| 1-512 | `minimal` |
| 513-1024 | `low` |
| 1025-8192 | `medium` |
| 8193-24576 | `high` |
| 24577+ | `xhigh` |

---

## 8. Complete Suffix → Result Matrix

### Claude Opus 4.6 (Levels: `["low", "medium", "high", "max"]`)

| Suffix | Config | Claude Provider Output | OpenAI Provider Output |
|:-------|:-------|:-----------------------|:-----------------------|
| `(low)` | ModeLevel, Level=low | `thinking.type="adaptive"` + `effort="low"` | `reasoning_effort="low"` |
| `(medium)` | ModeLevel, Level=medium | `thinking.type="adaptive"` + `effort="medium"` | `reasoning_effort="medium"` |
| `(high)` | ModeLevel, Level=high | `thinking.type="adaptive"` + `effort="high"` | `reasoning_effort="high"` |
| `(max)` | ModeLevel, Level=max | `thinking.type="adaptive"` + `effort="max"` | `reasoning_effort="high"` (clamped) |
| `(xhigh)` | ModeLevel → clamp to high | `thinking.type="adaptive"` + `effort="high"` | `reasoning_effort="high"` |
| `(minimal)` | ModeLevel → clamp to low | `thinking.type="adaptive"` + `effort="low"` | `reasoning_effort="low"` |
| `(none)` | ModeNone, Budget=0 | `thinking.type="disabled"` | `reasoning_effort="none"` |
| `(auto)` | ModeAuto → convert to medium | `thinking.type="adaptive"` + `effort="medium"` | `reasoning_effort="medium"` |
| `(8192)` | ModeBudget, Budget=8192 | `thinking.type="enabled"` + `budget_tokens=8192` | `reasoning_effort="medium"` |
| `(512)` | ModeBudget → clamp to 1024 | `thinking.type="enabled"` + `budget_tokens=1024` | `reasoning_effort="low"` |
| `(0)` | ModeNone, Budget=0 | `thinking.type="disabled"` | `reasoning_effort="none"` |

### Claude Sonnet 4.6 (Levels: `["low", "medium", "high"]`)

| Suffix | Config | Claude Provider Output | OpenAI Provider Output |
|:-------|:-------|:-----------------------|:-----------------------|
| `(low)` | ModeLevel, Level=low | `thinking.type="adaptive"` + `effort="low"` | `reasoning_effort="low"` |
| `(medium)` | ModeLevel, Level=medium | `thinking.type="adaptive"` + `effort="medium"` | `reasoning_effort="medium"` |
| `(high)` | ModeLevel, Level=high | `thinking.type="adaptive"` + `effort="high"` | `reasoning_effort="high"` |
| `(max)` | ModeLevel → clamp to high | `thinking.type="adaptive"` + `effort="high"` | `reasoning_effort="high"` |
| `(8192)` | ModeBudget, Budget=8192 | `thinking.type="enabled"` + `budget_tokens=8192` | `reasoning_effort="medium"` |
| `(none)` | ModeNone, Budget=0 | `thinking.type="disabled"` | `reasoning_effort="none"` |

### Claude 4.5 Sonnet/Opus (Budget-only, ZeroAllowed=true)

| Suffix | Config | Claude Provider Output | OpenAI Provider Output |
|:-------|:-------|:-----------------------|:-----------------------|
| `(high)` | ModeLevel → Level→Budget: 24576 | `thinking.type="enabled"` + `budget_tokens=24576` | `reasoning_effort="high"` |
| `(medium)` | ModeLevel → Level→Budget: 8192 | `thinking.type="enabled"` + `budget_tokens=8192` | `reasoning_effort="medium"` |
| `(low)` | ModeLevel → Level→Budget: 1024 | `thinking.type="enabled"` + `budget_tokens=1024` | `reasoning_effort="low"` |
| `(8192)` | ModeBudget, Budget=8192 | `thinking.type="enabled"` + `budget_tokens=8192` | `reasoning_effort="medium"` |
| `(none)` | ModeNone, Budget=0 | `thinking.type="disabled"` | `reasoning_effort="none"` |
| `(0)` | ModeNone, Budget=0 | `thinking.type="disabled"` | `reasoning_effort="none"` |

### Claude 4 / 4.1 (Budget-only, ZeroAllowed=false)

| Suffix | Config | Claude Provider Output | OpenAI Provider Output |
|:-------|:-------|:-----------------------|:-----------------------|
| `(high)` | ModeLevel → Level→Budget: 24576 | `thinking.type="enabled"` + `budget_tokens=24576` | `reasoning_effort="high"` |
| `(medium)` | ModeLevel → Level→Budget: 8192 | `thinking.type="enabled"` + `budget_tokens=8192` | `reasoning_effort="medium"` |
| `(none)` | ModeNone → Budget clamp to Min=1024 | `thinking.type="enabled"` + `budget_tokens=1024` | `reasoning_effort="low"` |
| `(0)` | ModeNone → Budget clamp to 1024 | `thinking.type="enabled"` + `budget_tokens=1024` | `reasoning_effort="low"` |

> [!WARNING]
> Claude 4 and 4.1 models have `ZeroAllowed=false`. Attempting to disable thinking via `(none)`
> or `(0)` results in the budget being clamped to `Min=1024`, not disabled.

---

## 9. Cross-Provider Conversion — Claude ↔ OpenAI

This is the most complex scenario: a request arrives in Anthropic format targeting a Claude model,
but the serving provider uses OpenAI format (ULW, openai-compatibility).

### Request Direction: Anthropic → OpenAI Provider

```
Client Request (Anthropic format):
{
    "thinking": {"type": "adaptive"},
    "output_config": {"effort": "high"},
    "messages": [...]
}
    │
    ▼ extractClaudeConfig(body)
ThinkingConfig{Mode: ModeLevel, Level: "high"}
    │
    ▼ ValidateConfig(config, modelInfo, from="claude", to="openai")
    - Model: claude-opus-4-6 (Hybrid)
    - ModeLevel + CapabilityHybrid → preserve (no auto-convert)
    - Level "high" ∈ model.Levels → valid
    - isSameProviderFamily("claude", "openai") = false
    → ThinkingConfig{Mode: ModeLevel, Level: "high"}
    │
    ▼ OpenAI Applier.Apply()
    - ModeLevel → set "reasoning_effort"
    │
    ▼
Final Request to OpenAI Provider:
{
    "model": "claude-opus-4-6",
    "reasoning_effort": "high",
    "messages": [...]
}
```

### OpenAI → Claude Conversion (request translator)

Source: `internal/translator/claude/openai/chat-completions/claude_openai_request.go`

When an OpenAI-format request targets a Claude model, `reasoning_effort` is converted:

| `reasoning_effort` | Model supports Levels? | Claude Output |
|:--------------------|:----------------------:|:-------------|
| `none` | ✅ | `thinking.type="disabled"` |
| `auto` | ✅ | `thinking.type="adaptive"` (no effort) |
| `low` | ✅ | `thinking.type="adaptive"` + `effort="low"` |
| `medium` | ✅ | `thinking.type="adaptive"` + `effort="medium"` |
| `high` | ✅ | `thinking.type="adaptive"` + `effort="high"` |
| `xhigh` / `max` | ✅ (supports max) | `thinking.type="adaptive"` + `effort="max"` |
| `xhigh` / `max` | ✅ (no max) | `thinking.type="adaptive"` + `effort="high"` |
| `low` | ❌ (budget-only) | `thinking.type="enabled"` + `budget_tokens=1024` |
| `medium` | ❌ (budget-only) | `thinking.type="enabled"` + `budget_tokens=8192` |
| `high` | ❌ (budget-only) | `thinking.type="enabled"` + `budget_tokens=24576` |

The `MapToClaudeEffort()` function handles level normalization:

| Generic Level | → Claude Effort | Notes |
|:--------------|:----------------|:------|
| `minimal` | `low` | Mapped down |
| `low` | `low` | Direct |
| `medium` | `medium` | Direct |
| `high` | `high` | Direct |
| `xhigh` | `max` (if supported) or `high` | Model-dependent |
| `max` | `max` (if supported) or `high` | Model-dependent |
| `auto` | `high` | Default fallback |

### Response Direction: Claude → OpenAI Format

Source: `internal/translator/claude/openai/chat-completions/claude_openai_response.go`

**Streaming conversion:**

| Claude Event | → OpenAI Format |
|:-------------|:-----------------|
| `thinking_delta` with `thinking` field | `delta.reasoning_content` |
| `text_delta` with `text` field | `delta.content` |
| `input_json_delta` | `delta.tool_calls[].function.arguments` |

**Non-streaming conversion:**

| Claude Content Block | → OpenAI Field |
|:---------------------|:---------------|
| `type: "thinking"` | `choices[0].message.reasoning` |
| `type: "text"` | `choices[0].message.content` |
| `type: "tool_use"` | `choices[0].message.tool_calls[]` |

---

## 10. Budget ↔ Level Conversion Tables

Source: `internal/thinking/convert.go`

### Level → Budget (`ConvertLevelToBudget`)

| Level | Budget (tokens) |
|:------|:---------------:|
| `none` | 0 |
| `auto` | -1 |
| `minimal` | 512 |
| `low` | 1024 |
| `medium` | 8192 |
| `high` | 24576 |
| `xhigh` | 32768 |
| `max` | 128000 |

### Budget → Level (`ConvertBudgetToLevel`)

| Budget Range | Level |
|:-------------|:------|
| -1 | `auto` |
| 0 | `none` |
| 1–512 | `minimal` |
| 513–1024 | `low` |
| 1025–8192 | `medium` |
| 8193–24576 | `high` |
| 24577+ | `xhigh` |

> [!NOTE]
> The `max` level is not produced by `ConvertBudgetToLevel`. It only exists as a
> `ConvertLevelToBudget` input mapping to 128000 tokens.

---

## 11. Anthropic Protocol — Body Config Extraction Scenarios

When a client sends a request in Anthropic format without a suffix, the config is extracted
from the request body via `extractClaudeConfig()`.

### Scenario A: Adaptive with explicit effort

```json
{
    "thinking": {"type": "adaptive"},
    "output_config": {"effort": "high"}
}
```

| Target Provider | Result |
|:----------------|:-------|
| Claude | `thinking.type="adaptive"` + `output_config.effort="high"` |
| OpenAI | `reasoning_effort="high"` |

### Scenario B: Adaptive without effort

```json
{
    "thinking": {"type": "adaptive"}
}
```

Extracts as empty config (no effort = no config). Body passes through unchanged.
Provider's default behavior applies.

### Scenario C: Enabled with budget

```json
{
    "thinking": {"type": "enabled", "budget_tokens": 16384}
}
```

| Target Provider | Result |
|:----------------|:-------|
| Claude (4.6 hybrid) | `thinking.type="enabled"` + `budget_tokens=16384` |
| Claude (4.5 budget-only) | `thinking.type="enabled"` + `budget_tokens=16384` |
| OpenAI | Budget 16384 → Level `high` → `reasoning_effort="high"` |

### Scenario D: Disabled

```json
{
    "thinking": {"type": "disabled"}
}
```

| Target Provider | Result |
|:----------------|:-------|
| Claude | `thinking.type="disabled"` |
| OpenAI (ZeroAllowed) | `reasoning_effort="none"` |

---

## 12. Key Source Files

| File | Lines | Purpose |
|:-----|:-----:|:--------|
| `internal/thinking/types.go` | 119 | ThinkingConfig, ThinkingMode, ThinkingLevel, ProviderApplier interface |
| `internal/thinking/suffix.go` | 148 | ParseSuffix, ParseSpecialSuffix, ParseLevelSuffix, ParseNumericSuffix |
| `internal/thinking/apply.go` | 521 | ApplyThinking orchestrator, extractThinkingConfig, format-specific extractors |
| `internal/thinking/validate.go` | 391 | ValidateConfig, auto-conversion, clamping, cross-family rules |
| `internal/thinking/convert.go` | 183 | ConvertLevelToBudget, ConvertBudgetToLevel, MapToClaudeEffort |
| `internal/thinking/provider/claude/apply.go` | 266 | Claude applier: adaptive + manual thinking |
| `internal/thinking/provider/openai/apply.go` | 117 | OpenAI applier: reasoning_effort |
| `internal/registry/model_registry.go` | 1228 | ModelInfo, ThinkingSupport, LookupModelInfo |
| `internal/registry/model_definitions_static_data.go` | 1029 | Static Claude model definitions |
| `internal/translator/claude/openai/chat-completions/claude_openai_request.go` | 360 | OpenAI→Claude request conversion |
| `internal/translator/claude/openai/chat-completions/claude_openai_response.go` | 508 | Claude→OpenAI response conversion (streaming + non-streaming) |

---

## 13. External References

| Resource | URL |
|:---------|:----|
| Anthropic Adaptive Thinking | `https://docs.anthropic.com/en/build-with-claude/adaptive-thinking` |
| Anthropic Extended Thinking | `https://docs.anthropic.com/en/build-with-claude/extended-thinking` |
| Anthropic Effort Parameter | `https://docs.anthropic.com/en/build-with-claude/effort` |
| Anthropic Messages API | `https://docs.anthropic.com/en/api/messages` |
