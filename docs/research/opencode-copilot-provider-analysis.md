# OpenCode GitHub Copilot Provider — Complete Analysis

> **Purpose**: LLM-friendly reference document for understanding how OpenCode's GitHub Copilot provider
> configures models, thinking/reasoning variants, API endpoint routing, and protocol-level differences
> across model families (Claude, GPT, Gemini, Grok).
>
> **Sources**: OpenCode source code (`packages/opencode/src/provider/`), models.dev API, CLIProxyAPIPlus
> Go implementation, GitHub issue anomalyco/opencode #11105.
>
> **Date**: 2026-03-06

---

## 1. Model Catalog (models.dev — GitHub Copilot Provider)

All models are free ($0 input/$0 output) when accessed through GitHub Copilot.

| Model ID | Family | Context | Output | Reasoning | Attachments | Release Date |
|----------|--------|---------|--------|-----------|-------------|--------------|
| `claude-opus-4.6` | claude-opus | 128,000 | 64,000 | yes | yes | 2026-02-05 |
| `claude-opus-4.5` | claude-opus | 128,000 | 32,000 | yes | yes | 2025-11-24 |
| `claude-opus-41` | claude-opus | 80,000 | 16,000 | yes | yes | 2025-08-05 |
| `claude-sonnet-4.6` | claude-sonnet | 128,000 | 32,000 | yes | yes | 2026-02-17 |
| `claude-sonnet-4.5` | claude-sonnet | 128,000 | 32,000 | yes | yes | 2025-09-29 |
| `claude-sonnet-4` | claude-sonnet | 128,000 | 16,000 | yes | yes | 2025-05-22 |
| `claude-haiku-4.5` | claude-haiku | 128,000 | 32,000 | yes | yes | 2025-10-15 |
| `gpt-5.4` | gpt | 400,000 | 128,000 | yes | no | 2026-03-05 |
| `gpt-5.3-codex` | gpt-codex | 400,000 | 128,000 | yes | no | 2026-02-24 |
| `gpt-5.2` | gpt | 128,000 | 64,000 | yes | yes | 2025-12-11 |
| `gpt-5.2-codex` | gpt-codex | 272,000 | 128,000 | yes | no | 2025-12-11 |
| `gpt-5.1` | gpt | 128,000 | 64,000 | yes | yes | 2025-11-13 |
| `gpt-5.1-codex` | gpt-codex | 128,000 | 128,000 | yes | no | 2025-11-13 |
| `gpt-5.1-codex-max` | gpt-codex | 128,000 | 128,000 | yes | yes | 2025-12-04 |
| `gpt-5.1-codex-mini` | gpt-codex | 128,000 | 128,000 | yes | no | 2025-11-13 |
| `gpt-5` | gpt | 128,000 | 128,000 | yes | yes | 2025-08-07 |
| `gpt-5-mini` | gpt-mini | 128,000 | 64,000 | yes | yes | 2025-08-13 |
| `gpt-4.1` | gpt | 64,000 | 16,384 | no | yes | 2025-04-14 |
| `gpt-4o` | gpt | 64,000 | 16,384 | no | yes | 2024-05-13 |
| `gemini-3.1-pro-preview` | gemini-pro | 128,000 | 64,000 | yes | yes | 2026-02-19 |
| `gemini-3-pro-preview` | gemini-pro | 128,000 | 64,000 | yes | yes | 2025-11-18 |
| `gemini-3-flash-preview` | gemini-flash | 128,000 | 64,000 | yes | yes | 2025-12-17 |
| `gemini-2.5-pro` | gemini-pro | 128,000 | 64,000 | no | yes | 2025-03-20 |
| `grok-code-fast-1` | grok | 128,000 | 64,000 | yes | no | 2025-08-27 |

---

## 2. API Endpoint Routing

### Decision Logic

OpenCode uses a two-endpoint system for GitHub Copilot:

- **`/chat/completions`** — OpenAI Chat Completions format (default for most models)
- **`/responses`** — OpenAI Responses API format (for GPT-5+ and Codex models)

There is **no native Anthropic `/v1/messages` support** in OpenCode's Copilot provider. All models,
including Claude, are wrapped in OpenAI-compatible format.

### Routing Function

Source: `packages/opencode/src/provider/provider.ts` (lines 52-62)

```typescript
function isGpt5OrLater(modelID: string): boolean {
    const match = /^gpt-(\d+)/.exec(modelID)
    if (!match) return false
    return Number(match[1]) >= 5
}

function shouldUseCopilotResponsesApi(modelID: string): boolean {
    return isGpt5OrLater(modelID) && !modelID.startsWith("gpt-5-mini")
}
```

### Custom Model Loader

Source: `packages/opencode/src/provider/provider.ts` (lines 163-172)

```typescript
"github-copilot": async () => ({
    autoload: false,
    async getModel(sdk, modelID) {
        if (sdk.responses === undefined && sdk.chat === undefined)
            return sdk.languageModel(modelID)
        return shouldUseCopilotResponsesApi(modelID)
            ? sdk.responses(modelID)
            : sdk.chat(modelID)
    },
    options: {},
})
```

### Endpoint Matrix

| Model Family | Endpoint | API Format |
|---|---|---|
| Claude (all versions) | `/chat/completions` | OpenAI Chat |
| GPT-4o, GPT-4.1 | `/chat/completions` | OpenAI Chat |
| GPT-5-mini | `/chat/completions` | OpenAI Chat |
| GPT-5, 5.1, 5.2, 5.3, 5.4 | `/responses` | OpenAI Responses |
| GPT-5.x-codex (all) | `/responses` | OpenAI Responses |
| Gemini (all) | `/chat/completions` | OpenAI Chat |
| Grok | `/chat/completions` | OpenAI Chat |

### Platform-Specific Endpoint Availability

Evidence from GitHub issue [anomalyco/opencode #11105](https://github.com/anomalyco/opencode/issues/11105):

The Copilot `/models` API returns a `supported_endpoints` array per model. This varies by platform:

| Platform | Claude Endpoints | Notes |
|---|---|---|
| github.com | `["/chat/completions", "/v1/messages"]` | Both available |
| GitHub Enterprise (GHE) | `["/chat/completions"]` only | No `/v1/messages` |

OpenCode does not use `/v1/messages` — it always routes through `/chat/completions`.

---

## 3. Thinking / Reasoning Configuration

### Constants

Source: `packages/opencode/src/provider/transform.ts` (lines 329-330)

```typescript
const WIDELY_SUPPORTED_EFFORTS = ["low", "medium", "high"]
const OPENAI_EFFORTS = ["none", "minimal", ...WIDELY_SUPPORTED_EFFORTS, "xhigh"]
```

### Variants by Provider × Model Family

The `ProviderTransform.variants()` function (transform.ts lines 332-679) returns different thinking
configurations depending on both the **SDK provider** and the **model family**.

#### Claude on Copilot (`@ai-sdk/github-copilot`)

Source: transform.ts lines 435-438

```typescript
if (model.id.includes("claude")) {
    return {
        thinking: { thinking_budget: 4000 },
    }
}
```

- Fixed 4,000 token thinking budget
- **No variants** (no low/medium/high/max)
- Uses snake_case `thinking_budget` (Copilot-specific)

#### Claude on Native Anthropic SDK (`@ai-sdk/anthropic`)

Source: transform.ts lines 517-549

**Adaptive models** (opus-4.6, sonnet-4.6):
```typescript
// Variants: low, medium, high, max
{
    thinking: { type: "adaptive" },
    effort: "low" | "medium" | "high" | "max",
}
```

**Non-adaptive Claude** (opus-4.5, sonnet-4.5, sonnet-4, haiku-4.5):
```typescript
// Variants: high, max
{
    thinking: {
        type: "enabled",
        budgetTokens: Math.min(16_000, Math.floor(model.limit.output / 2 - 1)),
    },
}
// max:
{
    thinking: {
        type: "enabled",
        budgetTokens: Math.min(31_999, model.limit.output - 1),
    },
}
```

#### Claude on AWS Bedrock (`@ai-sdk/amazon-bedrock`)

Source: transform.ts lines 551-595

**Adaptive models:**
```typescript
{
    reasoningConfig: {
        type: "adaptive",
        maxReasoningEffort: "low" | "medium" | "high" | "max",
    },
}
```

**Non-adaptive:**
```typescript
// high:
{ reasoningConfig: { type: "enabled", budgetTokens: 16000 } }
// max:
{ reasoningConfig: { type: "enabled", budgetTokens: 31999 } }
```

#### GPT-5 on Copilot (`@ai-sdk/github-copilot`)

Source: transform.ts lines 440-454

```typescript
// Base models (gpt-5, gpt-5-mini): ["low", "medium", "high"]
// Advanced models (5.1-codex-max, 5.2, 5.3): ["low", "medium", "high", "xhigh"]
{
    reasoningEffort: "low" | "medium" | "high" | "xhigh",
    reasoningSummary: "auto",
    include: ["reasoning.encrypted_content"],
}
```

#### GPT-5 on Native OpenAI (`@ai-sdk/openai`)

Source: transform.ts lines 486-515

```typescript
// Codex 5.2/5.3: ["low", "medium", "high", "xhigh"]
// GPT-5/5.x: adds "minimal" prefix, "none" if released after 2025-11-13,
//            "xhigh" if released after 2025-12-04
{
    reasoningEffort: "none" | "minimal" | "low" | "medium" | "high" | "xhigh",
    reasoningSummary: "auto",
    include: ["reasoning.encrypted_content"],
}
```

#### Gemini on Copilot (`@ai-sdk/github-copilot`)

Source: transform.ts lines 431-433

```typescript
if (model.id.includes("gemini")) {
    return {}  // No thinking variants on Copilot
}
```

#### Gemini on Native Google (`@ai-sdk/google`)

Source: transform.ts lines 597-632

**Gemini 2.5:**
```typescript
// Variants: high, max
{ thinkingConfig: { includeThoughts: true, thinkingBudget: 16000 } }   // high
{ thinkingConfig: { includeThoughts: true, thinkingBudget: 24576 } }   // max
```

**Gemini 3.x:**
```typescript
// Variants: low, high
{ thinkingConfig: { includeThoughts: true, thinkingLevel: "low" | "high" } }
```

**Gemini 3.1:**
```typescript
// Variants: low, medium, high
{ thinkingConfig: { includeThoughts: true, thinkingLevel: "low" | "medium" | "high" } }
```

#### Grok

Source: transform.ts lines 352-364

**grok-3-mini only:**
```typescript
// OpenRouter: { reasoning: { effort: "low" | "high" } }
// Others:     { reasoningEffort: "low" | "high" }
```

All other Grok models return empty (no thinking).

### Complete Thinking Matrix

| Model | Copilot Thinking | Native SDK Thinking |
|---|---|---|
| claude-opus-4.6 | `thinking_budget: 4000` (fixed) | adaptive: low/medium/high/max |
| claude-sonnet-4.6 | `thinking_budget: 4000` (fixed) | adaptive: low/medium/high/max |
| claude-opus-4.5 | `thinking_budget: 4000` (fixed) | budgetTokens: 16K (high), 31999 (max) |
| claude-sonnet-4.5 | `thinking_budget: 4000` (fixed) | budgetTokens: 16K (high), 31999 (max) |
| claude-sonnet-4 | `thinking_budget: 4000` (fixed) | budgetTokens: 16K (high), 31999 (max) |
| claude-haiku-4.5 | `thinking_budget: 4000` (fixed) | budgetTokens: 16K (high), 31999 (max) |
| gpt-5 | reasoningEffort: low/med/high | +none, +minimal |
| gpt-5.1-codex-max | reasoningEffort: low/med/high/xhigh | +none |
| gpt-5.2 | reasoningEffort: low/med/high/xhigh | +none, +minimal, +xhigh |
| gpt-5.2-codex | reasoningEffort: low/med/high/xhigh | +none |
| gpt-5.3-codex | reasoningEffort: low/med/high/xhigh | +none |
| gemini-3-pro | none | thinkingLevel: low/high |
| gemini-3.1-pro | none | thinkingLevel: low/medium/high |
| gemini-2.5-pro | none | thinkingBudget: 16K/24576 |
| grok-code-fast-1 | none | none |

---

## 4. Protocol-Level Differences

### 4.1 Tool Call ID Normalization (Claude-Specific)

Source: `packages/opencode/src/provider/transform.ts` (lines 74-89)

```typescript
if (model.api.id.includes("claude")) {
    part.toolCallId = part.toolCallId.replace(/[^a-zA-Z0-9_-]/g, "_")
}
```

Claude rejects tool call IDs containing non-alphanumeric characters. OpenCode normalizes them
by replacing invalid characters with underscores.

### 4.2 Cache Control (Provider-Specific Headers)

Source: `packages/opencode/src/provider/transform.ts` (lines 174-212)

| Provider | Cache Header | Value |
|---|---|---|
| Copilot (`@ai-sdk/github-copilot`) | `copilot_cache_control` | `{ type: "ephemeral" }` |
| Anthropic (`@ai-sdk/anthropic`) | `cacheControl` | `{ type: "ephemeral" }` |
| OpenRouter | `cacheControl` | `{ type: "ephemeral" }` |
| AWS Bedrock | `cachePoint` | `{ type: "default" }` |
| OpenAI-compatible | `cache_control` | `{ type: "ephemeral" }` |

Caching is applied only to Anthropic/Claude models (transform.ts lines 255-265).

### 4.3 Reasoning State Preservation (Copilot-Specific)

Source: `packages/opencode/src/provider/sdk/copilot/chat/openai-compatible-chat-language-model.ts`
(lines 225-241)

Copilot extends the standard OpenAI Chat Completions response with two reasoning fields:

- **`reasoning_text`** — Human-readable thinking process (displayed to user)
- **`reasoning_opaque`** — Encrypted multi-turn reasoning state token

```typescript
providerMetadata: choice.message.reasoning_opaque
    ? { copilot: { reasoningOpaque: choice.message.reasoning_opaque } }
    : undefined
```

These are Copilot-specific extensions not present in the standard OpenAI API.

### 4.4 Assistant Content Flattening

Source: `CLIProxyAPIPlus/internal/runtime/executor/github_copilot_executor.go` (lines 578-616)

Copilot expects assistant message content as a **string**, not an array. Claude's native format
uses array content blocks. The proxy flattens these before sending:

```
Claude format:  messages[i].content = [{ type: "text", text: "..." }]   // ARRAY
OpenAI format:  messages[i].content = "..."                              // STRING
```

The `flattenAssistantContent()` function joins all text parts into a single string for assistant
messages, skipping messages that contain non-text blocks (tool_use, thinking).

### 4.5 Request Headers

Source: `CLIProxyAPIPlus/internal/runtime/executor/github_copilot_executor.go` (lines 477-490)

```
Authorization:  Bearer {copilot_api_token}
Content-Type:   application/json
Accept:         application/json
User-Agent:     opencode/0.1.0
Openai-Intent:  conversation-edits
X-Initiator:    user | agent     (dynamic, based on last message role)
```

---

## 5. Architecture: How Variants Are Generated

The thinking configuration shown in model definitions is **not stored in models.dev**. It is computed
at runtime by OpenCode through the following pipeline:

```
models.dev API (base model data: limits, capabilities, cost)
         │
         ▼
ProviderTransform.variants()  (transform.ts:332-679)
    Switch on model.api.npm (SDK provider package)
    Then check model.id (model family)
    Returns: Record<string, Record<string, any>>
         │
         ▼
Provider.ts merges variants with model definition
    Final model object with computed thinking config
```

The `variants` field in models.dev schema is optional (models.ts line 69) and typically empty.
The actual thinking configurations come from the `ProviderTransform.variants()` switch statement,
which maps each SDK provider + model combination to specific reasoning parameters.

---

## 6. SDK Provider Mapping

Source: `packages/opencode/src/provider/provider.ts` (lines 88-111)

| NPM Package | Provider | Notes |
|---|---|---|
| `@ai-sdk/github-copilot` | GitHub Copilot | Custom `createGitHubCopilotOpenAICompatible` |
| `@ai-sdk/anthropic` | Anthropic (direct) | Native Messages API |
| `@ai-sdk/openai` | OpenAI (direct) | Native API |
| `@ai-sdk/azure` | Azure OpenAI | Azure-hosted models |
| `@ai-sdk/google` | Google AI | Gemini models |
| `@ai-sdk/google-vertex` | Google Vertex AI | Enterprise Gemini |
| `@ai-sdk/google-vertex/anthropic` | Vertex Anthropic | Claude on Vertex |
| `@ai-sdk/amazon-bedrock` | AWS Bedrock | Claude/Nova on AWS |
| `@ai-sdk/xai` | xAI | Grok models |
| `@openrouter/ai-sdk-provider` | OpenRouter | Multi-provider gateway |
| `@ai-sdk/gateway` | AI Gateway | Unified gateway |
| `@ai-sdk/openai-compatible` | Generic OpenAI-compat | Custom endpoints |

---

## 7. Key Source Files

### OpenCode (TypeScript)

| File | Lines | Purpose |
|---|---|---|
| `packages/opencode/src/provider/provider.ts` | 1360 | Endpoint routing, SDK loading, custom loaders |
| `packages/opencode/src/provider/transform.ts` | ~880 | Thinking variants, message normalization, caching |
| `packages/opencode/src/provider/models.ts` | ~130 | models.dev API loading and caching |
| `packages/opencode/src/provider/sdk/copilot/copilot-provider.ts` | ~100 | Copilot SDK factory |
| `packages/opencode/src/provider/sdk/copilot/chat/openai-compatible-chat-language-model.ts` | ~780 | Chat completions implementation |
| `packages/opencode/src/provider/sdk/copilot/responses/openai-responses-language-model.ts` | ~1200 | Responses API implementation |
| `packages/opencode/src/plugin/copilot.ts` | ~330 | OAuth flow, auth headers |

### CLIProxyAPIPlus (Go)

| File | Lines | Purpose |
|---|---|---|
| `internal/runtime/executor/github_copilot_executor.go` | 1355 | Endpoint routing, format translation, streaming |
| `internal/registry/model_definitions.go` | 799 | Static model definitions with endpoint support |
| `internal/registry/model_registry.go` | 1228 | Dynamic model registry with reference counting |
| `internal/auth/copilot/copilot_auth.go` | 281 | OAuth token handling, model discovery |

---

## 8. External References

| Resource | URL |
|---|---|
| models.dev API | `https://models.dev/api.json` |
| Copilot API Base | `https://api.githubcopilot.com` |
| Chat Completions | `https://api.githubcopilot.com/chat/completions` |
| Responses API | `https://api.githubcopilot.com/responses` |
| Models Discovery | `https://api.githubcopilot.com/models` |
| OpenCode Client ID | `Ov23li8tweQw6odWQebz` |
| GHE Endpoint Issue | [anomalyco/opencode #11105](https://github.com/anomalyco/opencode/issues/11105) |
