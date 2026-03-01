# Fork-Specific Features & Enhancements

This document tracks all features unique to the [anilcancakir/CLIProxyAPIPlus](https://github.com/anilcancakir/CLIProxyAPIPlus) fork that are **not present** in the upstream [router-for-me/CLIProxyAPIPlus](https://github.com/router-for-me/CLIProxyAPIPlus).

> [!NOTE]
> This fork merges improvements from two community forks:
> - [lemon07r/CLIProxyAPIPlus](https://github.com/lemon07r/CLIProxyAPIPlus) — Copilot Claude routing, Antigravity anti-fingerprinting, translator fixes
> - [KooshaPari/cliproxyapi-plusplus](https://github.com/KooshaPari/cliproxyapi-plusplus) — SDK enhancements, sticky routing, fallback models, CLI tooling

---

## GitHub Copilot — Claude & GPT-5 Support

Tri-state endpoint routing enables Claude and GPT-5 models through GitHub Copilot:

| Model Type | Endpoint | Format |
|:-----------|:---------|:-------|
| Claude (Sonnet, Opus) | `/v1/messages` | Native Anthropic |
| GPT-5 / Codex | `/responses` | Responses API |
| Legacy (GPT-4o, etc.) | `/chat/completions` | OpenAI |

Key functions: `isCopilotClaudeModel()`, `isGPT5Model()`, `getEndpointPath()`

### Unlimited Premium Access

Headers are configured to match the official VS Code Copilot agent:

- `X-Initiator: agent` — always set for unlimited premium request access
- `X-Github-Api-Version: 2025-10-01`
- Dynamic `VScode-SessionId` and `VScode-MachineId` per request
- `copilot-chat/0.37.2026013101` plugin version

### Claude Thinking Budget

`normalizeCopilotClaudeThinking()` automatically configures thinking budgets for Sonnet 3.7+ models, with `anthropic-beta` header injection for interleaved thinking support.

### SDK Prefix Handling

Model names with `copilot-` prefix are automatically stripped in:
- `sdk/api/handlers/handlers.go` — provider resolution fallback
- `sdk/api/handlers/openai/endpoint_compat.go` — registry lookup
- `sdk/api/handlers/openai/openai_handlers.go` — responses routing

**Source:** Lemon fork patches 001, 002, 007
**Files:** `internal/runtime/executor/github_copilot_executor.go`, `sdk/api/handlers/openai/`

---

## Antigravity — Anti-Fingerprinting

Dynamic version fetching and per-account User-Agent rotation to prevent account flagging.

### Dynamic Version Fetching

```
URL:     https://antigravity-auto-updater-974169037036.us-central1.run.app
Cache:   12 hours (antigravityVersionRefresh)
Fallback: last known version on error
```

`fetchAntigravityVersion()` calls the auto-updater API, parses the version via regex, and caches the result. The `compareVersions()` helper ensures semver-correct comparisons.

### Per-Account User-Agent

Each account gets a stable, deterministic User-Agent based on its `auth.ID` hash:

- UA format: `antigravity/{version} {platform}`
- Platform pool: `darwin/arm64`, `linux/amd64`, etc.
- Version downgrade protection via `antigravityAccountVersions` map
- Account-locked: same account always gets the same UA string

### Session ID Hardening

`geminiToAntigravity()` now accepts `authID` and generates session IDs in the format:
```
-{uuid}:{model}:{project}:seed-{hex16}
```

This prevents cross-account correlation by salting session IDs with the auth identity.

**Source:** Lemon fork patch 006
**Files:** `internal/runtime/executor/antigravity_executor.go`

---

## Antigravity — Quota Tracking

Proactive quota awareness with reason-based backoff and model-level rate limit isolation.

### Quota Checker

```
Endpoint: https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels
Cache:    5-minute TTL per account (sync.RWMutex)
Handles:  403 Forbidden → IsForbidden=true
```

`AntigravityQuotaChecker` satisfies the `QuotaChecker` interface. Returns `QuotaData` containing `[]ModelQuota` — each with `Name`, `RemainingFraction float64`, `ResetTime time.Time`, and `DisplayName`. No background polling; cache is refreshed on-demand.

### Rate Limiter

`AntigravityRateLimiter` enforces reason-based backoff with model-level isolation (`accountID:modelName` keys):

| Reason | Strategy | Delays |
|:-------|:---------|:-------|
| `QuotaExhausted` | Exponential | 60 → 300 → 1800 → 7200s |
| `RateLimitExceeded` | Fixed account-wide | 5s |
| `ModelCapacityExhausted` | Progressive | 5 / 10 / 15s |
| `ServerError` | Fixed, no failure increment | 8s |

`ServerError` (5xx) does not pollute `QuotaExhausted` failure count — preventing false exponential backoff from transient errors. Exhausting one model (e.g. `gemini-2.5-pro`) does not block other models on the same account.

### Executor Integration

The `antigravity_executor.go` checks `IsRateLimited()` before dispatch, calls `rateLimiter.ParseFromError()` on 429/5xx responses, and calls `MarkSuccess()` on success. `FetchAntigravityModels` extracts and caches `quotaInfo.remainingFraction` + `resetTime` from model discovery.

### Management Endpoint

`GET /v0/management/antigravity-quota` returns a JSON array of per-account quota state:

```json
[
  {
    "account_id": "...",
    "email": "...",
    "models": [
      {
        "name": "gemini-2.5-pro",
        "remaining_fraction": 0.75,
        "remaining_percent": 75,
        "reset_time": "..."
      }
    ]
  }
]
```

**Files:** `internal/auth/antigravity/quota_checker.go`, `internal/auth/antigravity/rate_limiter.go`, `internal/runtime/executor/antigravity_executor.go`, `internal/api/handlers/management/antigravity_quota.go`
---

## Claude Code — Request Cloaking & Prompt Caching

Direct Anthropic API access with identity cloaking, automated prompt caching, and TLS fingerprint bypass. Any client (curl, OpenAI SDK, Roo-Code, etc.) is transparently disguised as the official Claude Code CLI.

### Request Cloaking

`ClaudeExecutor` rewrites outgoing requests to match the official Claude Code CLI v2.1.63 wire format. Clients that are not already identified as Claude Code get a full identity transplant before the request reaches Anthropic.

**Billing header injection**

Every request receives an `x-anthropic-billing-header` containing the CLI version, entrypoint, and a `cch` hash — required for Claude Code subscription billing to apply:

```
x-anthropic-billing-header: {"version":"2.1.63","entrypoint":"cli","cch":"<hash>"}
```

**Claude Code agent system prompt**

When cloaking is active, a system prompt is prepended identifying the agent:

```
You are a Claude agent, built on Anthropic's Claude Agent SDK.
```

**Fake user_id generation**

Each request is assigned a synthetic user identity matching the Claude Code session format:

```
user_[64hex]_account_[uuid]_session_[uuid]
```

With `cache-user-id: true`, the generated ID is stable per configured API key to improve prompt cache hit rates.

**Sensitive word obfuscation**

Words listed under `sensitive-words` in the cloak config are obfuscated via zero-width character insertion before the request is sent, preventing keyword-based filtering.

**Header emulation**

The following headers are set to match Claude Code 2.1.63 / `@anthropic-ai/sdk` 0.74.0:

| Header | Value |
|:-------|:------|
| `User-Agent` | `claude-cli/2.1.63 (external, cli)` |
| `X-Stainless-Package-Version` | `0.74.0` |
| `X-Stainless-Timeout` | `600` |

Non-Claude clients have their User-Agent forcibly replaced to prevent identity leaks. Tool name prefix is set to an empty string to match real Claude Code behavior.

**Cloak modes**

| Mode | Behavior |
|:-----|:---------|
| `auto` (default) | Cloak only non-Claude clients |
| `always` | Cloak all clients unconditionally |
| `never` | Disable cloaking entirely |

**Strict mode**

When `strict-mode: true`, all user-supplied system messages are stripped and replaced with the Claude Code identity prompt, ensuring a consistent identity regardless of client input.

### Automated Prompt Caching

Prompt caching cuts repeated inference costs by up to 90% — cached reads are billed at 0.1x the base token price.

`ClaudeExecutor` auto-injects `cache_control: {"type": "ephemeral"}` breakpoints whenever a request arrives with no existing cache markers. Injection follows a three-level priority order, stopping once a valid breakpoint is placed:

1. **Tools** — last tool in the tools array
2. **System** — last element of the system prompt array
3. **Messages** — second-to-last user turn

Anthropic permits up to 4 cache breakpoints per request; the executor respects this limit. System prompts receive `ttl: "1h"` cache control for longer-lived caching.

For full caching semantics, see the [Anthropic prompt caching docs](https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching).

### TLS Fingerprint Bypass

Standard Go TLS produces a fingerprint that differs from browser and Node.js clients. The `utls_transport.go` replaces the default HTTP client with a `utls`-backed transport:

- **Fingerprint:** `tls.HelloChrome_Auto` — closely matches Node.js/OpenSSL used by the real Claude Code CLI
- **Connection pooling:** HTTP/2 per-host connection reuse for reduced latency
- **Proxy support:** `proxy-url` in config is forwarded to the custom transport

### OAuth Authentication

`cliproxyctl login --provider claude` runs an OAuth2 PKCE flow:

1. A local callback server (`oauth_server.go`) opens a browser to the Anthropic authorization URL
2. The authorization code is captured on the redirect
3. Tokens are exchanged and stored as `claude-{email}.json`

**Auth file format:**

```json
{
  "access_token": "sk-ant-oat...",
  "refresh_token": "...",
  "email": "user@example.com",
  "expired": false
}
```

Token refresh is handled by `RefreshTokens()`, which exchanges the refresh token for a new `sk-ant-oat` access token without user interaction.

### Configuration

```yaml
claude-api-key:
  - api-key: "sk-ant-..."
    base-url: "https://api.anthropic.com"  # optional
    priority: 10
    cloak:
      mode: auto        # auto | always | never
      strict-mode: false
      sensitive-words: ["secret", "password"]
      cache-user-id: true

claude-header-defaults:
  user-agent: "claude-cli/2.1.63 (external, cli)"
  package-version: "0.74.0"
  runtime-version: "v24.3.0"
  timeout: "600"
```

**Files:**

| File | Purpose |
|:-----|:--------|
| `internal/runtime/executor/claude_executor.go` | Core executor with cloaking and prompt caching |
| `internal/runtime/executor/cloak_utils.go` | Cloak mode detection and user ID generation |
| `internal/auth/claude/utls_transport.go` | uTLS Chrome fingerprint transport |
| `internal/auth/claude/oauth_server.go` | OAuth callback server |
| `internal/auth/claude/anthropic.go` | Auth data structures |
| `sdk/api/handlers/claude/code_handlers.go` | `/v1/messages` endpoint handler |
| `sdk/api/handlers/claude/request_sanitize.go` | Request cleanup |
| `internal/config/config.go` | `ClaudeKey`, `CloakConfig`, `ClaudeHeaderDefaults` types |

### Quota Threshold Fallback

Per-model quota utilization thresholds that trigger fast 429 errors **before** the API call is made. When a model's 5-hour utilization exceeds the configured threshold, the executor returns a `quotaThresholdError` with `RetryAfter` set to the quota reset time. The conductor's existing fallback mechanism picks up the 429 and routes to alternative providers (Antigravity, Copilot) via priority routing.

**Pre-execution check**

`checkQuotaThreshold()` runs immediately after `baseModel` parsing in both `Execute()` and `ExecuteStream()` — before any HTTP work. It reads `GetCachedQuota()` (thread-safe, read-only via `RLock`) and compares `FiveHour.Utilization` against the configured threshold. Cold starts (no cached quota) pass through without blocking.

**Wildcard matching**

Model patterns support `filepath.Match` wildcards (e.g., `claude-opus-*: 80`). The first matching pattern wins.

**Error interface**

`quotaThresholdError` implements the same interface contract as `modelCooldownError`:

| Method | Behavior |
|:-------|:---------|
| `Error()` | JSON with `code: "quota_threshold_exceeded"`, utilization, threshold, reset info |
| `StatusCode()` | `429 Too Many Requests` |
| `RetryAfter()` | `*time.Duration` — `time.Until(resetsAt)`, clamped ≥ 0 |
| `Headers()` | `Content-Type: application/json`, `Retry-After: <seconds>` |

The conductor's `retryAfterFromError()` detects the `RetryAfter() *time.Duration` interface and auto-sets `NextRetryAfter` for recovery — no `StatusDisabled`, no permanent blocks.

**Configuration**

```yaml
quota-exceeded:
  claude-quota-threshold:
    claude-opus-4-5-20251101: 80      # Fail over Opus at 80% utilization
    claude-sonnet-4-5-20250929: 95    # Fail over Sonnet at 95% utilization
```

Values are 0–100 utilization percentages matching `FiveHour.Utilization` from the Claude quota API. Hot-reload safe — thresholds are read from the live `cfg` pointer at call time, not at init.

**Files:**

| File | Purpose |
|:-----|:--------|
| `internal/runtime/executor/claude_quota_threshold.go` | `quotaThresholdError` struct (110 LOC) |
| `internal/runtime/executor/claude_executor.go` | `checkQuotaThreshold()` helper + `Execute`/`ExecuteStream` integration |
| `internal/config/config.go` | `ClaudeQuotaThresholds map[string]float64` field |
| `internal/runtime/executor/claude_executor_test.go` | Threshold TDD tests (above/below, nil cache, wildcard) |
| `internal/config/config_test.go` | Config parsing TDD tests |

---

## Translator Improvements

### Thinking Signature Fix (Claude via Antigravity)

Invalid thinking block signatures (from different sessions/models) are silently dropped instead of causing 400 errors. The `enableThoughtTranslate` flag was removed — thinking blocks are always processed, and unsigned blocks are dropped gracefully.

Non-stream responses now also cache signatures via `cache.CacheSignature()`, matching stream behavior.

**Files:** `internal/translator/antigravity/claude/antigravity_claude_request.go`, `antigravity_claude_response.go`

### Consecutive Turn Merging (Gemini)

Multiple consecutive messages with the same role are merged into a single message before sending to the Gemini API, which enforces strict role alternation.

**File:** `internal/translator/antigravity/gemini/antigravity_gemini_request.go`

### Assistant Prefill (Gemini)

Trailing assistant messages without function calls are detected and converted into a synthetic user message with `Continue from: {text}` prefix, allowing the model to resume generation.

**File:** `internal/translator/antigravity/gemini/antigravity_gemini_request.go`

### Streaming Tool Call Deltas (Claude → OpenAI)

The Claude-to-OpenAI response translator now supports real-time `tool_use` and `text` deltas in SSE streams. Tool call arguments are properly accumulated and emitted across chunks.

`convertPlainClaudeResponseToOpenAI()` handles non-stream responses.

**Source:** Lemon fork patches 003, 004, 005, 008
**Files:** `internal/translator/claude/openai/chat-completions/claude_openai_response.go`

---

## SDK Enhancements

### Sticky Session Routing

`StickyRoundRobinSelector` pins users to the same credential based on `X-Session-Key` header:

- Session key present → reuse pinned credential (or pin a new one via round-robin)
- No session key → default to `FillFirstSelector`
- Memory bounded: 8192 session limit with LRU eviction

**File:** `sdk/cliproxy/auth/selector.go`

### Conductor Fallback Models

`executeWithFallback()` pattern enables automatic model degradation:

1. Primary model fails or has no available auths
2. System checks `fallback_models` context key
3. Automatically switches to next model in the chain
4. Returns `503 Service Unavailable` with `Retryable: true` only when all options exhausted

**File:** `sdk/cliproxy/auth/conductor.go`

### Claude Request Sanitization

`sanitizeClaudeRequest()` removes placeholder fields from tool schemas:

- Strips `_` and `reason` fields with known placeholder descriptions
- Removes those fields from `required` arrays
- Prevents Claude API rejections from client-injected dummy properties

**File:** `sdk/api/handlers/claude/request_sanitize.go`

### OpenAI Images Handler

Cross-provider image generation: accepts OpenAI DALL-E format, routes to either OpenAI or Gemini Imagen:

- OpenAI `size` → Gemini `aspect_ratio` conversion
- OpenAI `n` → Gemini `sampleCount` mapping
- Gemini base64 response → OpenAI `data: [{url}]` format

**File:** `sdk/api/handlers/openai/openai_images_handlers.go`

### Extended Config Type Exports

SDK consumers can now access these types: `RoutingConfig`, `AmpModelMapping`, `QuotaExceeded`, `GeminiModel`, `ClaudeModel`, `CodexModel`, `KiroKey`, `KiroFingerprintConfig`, `ClaudeHeaderDefaults`, `AmpUpstreamAPIKeyEntry`.

**File:** `sdk/config/config.go`

### X-Session-Key Forwarding

Session key is extracted from the request header and forwarded to the metadata map for sticky routing.

**File:** `sdk/api/handlers/handlers.go`

### Enhanced Error Types

`Error` struct now includes `Retryable bool` and `HTTPStatus int` fields, enabling the conductor's fallback mechanism to make retry decisions based on error semantics.

**File:** `sdk/cliproxy/auth/types.go`

### Stream Cancellation

`StreamResult` includes `Cancel context.CancelFunc` to prevent connection leaks in WebSocket-based executors (Codex WS).

**File:** `sdk/cliproxy/executor/types.go`

---

## OAuth Provider Priority

When multiple OAuth providers register the same model alias, the system previously fell back to provider ordering by client count — not user-controllable. This feature lets you assign explicit integer priorities at two levels.

### Priority Resolution Hierarchy

1. **Account-level** — `"priority"` field in the JSON auth file (highest; overrides everything)
2. **Provider-level** — `oauth-provider-priority` map in YAML config (fallback when no JSON priority)
3. **Default 0** — applied when neither is set

Account-level priority wins unconditionally. An explicit `"priority": 0` in a JSON file overrides a non-zero provider config value (zero-value distinction is preserved). Negative values are allowed for explicit de-prioritization below the default.

### Config (Provider-Level)

```yaml
# Higher value = higher preference. Default: 0.
# Account-level "priority" in JSON auth files overrides these.
oauth-provider-priority:
  claude: 10        # Prefer direct Claude OAuth over antigravity
  antigravity: 0    # Default priority
  kiro: 5           # Medium priority
```

### JSON Auth File (Account-Level)

```json
{
  "type": "claude",
  "email": "user@example.com",
  "priority": 15
}
```

Missing `oauth-provider-priority` in config leaves the map nil — no panic, full backward compatibility.

**Files:** `internal/config/config.go`, `sdk/auth/filestore.go`, `internal/watcher/synthesizer/file.go`

---

## Kilo AI Provider

Full Kilo AI (OpenRouter) integration:

- **Dynamic Model Discovery**: Fetches models from `api.kilo.ai/api/openrouter/models`, filtering for curated free models
- **Device Flow Auth**: OAuth device flow login with organization selection
- **Dedicated Executor**: `NewKiloExecutor` with `X-Kilocode-OrganizationID` header injection
- **Token Persistence**: Stores token, org ID, and email in `KiloTokenStorage`

**Files:** `internal/runtime/executor/kilo_executor.go`, `sdk/auth/kilo.go`

---

## Kiro Web Search

MCP (Model Context Protocol) based web search for Kiro (AWS CodeWhisperer):

- `tools/list` and `web_search` call implementation
- Lock-free caching via `atomic.Pointer[sync.Once]`
- Exponential backoff retries (AWS GAR patterns)

**File:** `internal/translator/kiro/claude/kiro_websearch_handler.go`

---

## Smart Routing

`POST /v1/routing/select` endpoint for intent-based model selection:

| Complexity | Routed Model |
|:-----------|:-------------|
| FAST | minimax-m2.5 |
| NORMAL | claude-sonnet-4.6 |
| COMPLEX | gpt-5.3-codex |
| HIGH_COMPLEX | gpt-5.3-codex-xhigh |

Clients request a model by intent/cost constraint instead of hardcoding model IDs.

**File:** `internal/api/handlers/management/routing_select.go`

---

## CLI Control Tool (cliproxyctl)

Standalone CLI utility for proxy management:

| Command | Description |
|:--------|:------------|
| `setup` | Interactive configuration wizard |
| `login` | OAuth flows (Gemini, Kiro) with `--no-browser` support |
| `doctor` | Diagnostic check: config validity, provider count, file existence |

Supports `--json` flag for machine-readable output.

**Files:** `cliproxyctl/main.go`, `cliproxyctl/setup.go`

---

## CI/CD & Infrastructure

### Automated Upstream Sync

GitHub Actions workflow (`.github/workflows/sync-and-build.yml`):

- Hourly upstream sync via cron (`0 */1 * * *`)
- Multi-arch Docker build (`linux/amd64`, `linux/arm64`)
- Auto-publish to DockerHub: `anilcancakir/cli-proxy-api-plus`
- Triggers on: push to main, patch changes, Go file changes

### Dockerfile Patch Mechanism

Builder stage applies `patches/*.patch` files in alphanumeric order via `git apply`:

```dockerfile
RUN for patch in $(ls patches/*.patch | sort); do
    git apply --check "$patch" && git apply "$patch"
done
```

Enables maintaining fork features as clean, reapplyable patches on top of synced upstream code.

### Utility Scripts

- `docker-auto-update.sh` — Automated Docker image update
- `setup-cron.sh` — Cron job setup for auto-updates

---

## Test Coverage

Additional test files not present in upstream:

| File | Coverage |
|:-----|:---------|
| `test/e2e_test.go` | Health checks, binary existence |
| `test/openai_websearch_translation_test.go` | Websearch MCP translation |
| `test/roo_kilo_login_integration_test.go` | Kilo/Roo-Code login flow |
| `sdk/access/manager_test.go` | Access manager |
| `sdk/api/handlers/claude/request_sanitize_test.go` | Claude tool sanitization |
| `sdk/api/handlers/openai/openai_images_handlers_test.go` | Image handler |
| `sdk/api/handlers/handlers_*_test.go` | Handler metadata, error responses |
| `sdk/config/config_test.go` | Config type exports |
| `internal/translator/openai/claude/openai_claude_response_test.go` | Claude response translation |
| `internal/config/config_test.go` | OAuthProviderPriority YAML parsing |
| `sdk/auth/filestore_test.go` | Priority parsing from JSON metadata |
| `internal/watcher/synthesizer/file_test.go` | Priority resolution scenarios |
| `internal/auth/antigravity/quota_checker_test.go` | Quota checker: caching, 403 handling, model parsing |
| `internal/auth/antigravity/rate_limiter_test.go` | Rate limiter: reason-based backoff, model isolation |
| `internal/auth/antigravity/integration_test.go` | End-to-end quota + rate limiter integration |
| `internal/runtime/executor/claude_executor_test.go` | Cloaking, prompt cache injection, user ID, quota threshold |

---

## Attribution

| Feature Set | Source Fork | Patches |
|:------------|:-----------|:--------|
| Copilot Claude/GPT-5 routing | [lemon07r](https://github.com/lemon07r/CLIProxyAPIPlus) | 001, 002, 007 |
| Antigravity anti-fingerprinting | [lemon07r](https://github.com/lemon07r/CLIProxyAPIPlus) | 006 |
| Thinking signature & translator fixes | [lemon07r](https://github.com/lemon07r/CLIProxyAPIPlus) | 003, 004, 005, 008 |
| SDK routing, fallbacks, sanitization | [KooshaPari](https://github.com/KooshaPari/cliproxyapi-plusplus) | — |
| Kilo provider, cliproxyctl, TUI | [KooshaPari](https://github.com/KooshaPari/cliproxyapi-plusplus) | — |
| Antigravity quota tracking | original (this fork) | — |
| Claude quota threshold fallback | original (this fork) | — |
