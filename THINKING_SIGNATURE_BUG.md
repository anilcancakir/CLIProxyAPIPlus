# Thinking Signature Bug — Root Cause Analysis & Fix Plan

> **Status**: Root cause confirmed, fix plan ready, awaiting implementation  
> **Date**: 2026-03-01  
> **Blocking**: Extended thinking in multi-turn conversations via OpenCode

---

## TL;DR

`generateBillingHeader()` uses `rand.Read()` + payload SHA256 to create a billing header
injected into system prompt. This changes every turn. Claude's thinking signatures are
cryptographically bound to the system prompt context. Different system prompt = invalid
signature = 400 error.

**Fix**: Strip historical thinking blocks from messages BEFORE `applyCloaking()` runs.
Claude accepts conversations without prior thinking blocks.

---

## Error Details

```
Error:    messages.3.content.0: Invalid 'signature' in 'thinking' block
HTTP:     400 Bad Request
Latency:  ~425-514ms (fast API validation reject)
Endpoint: POST /v1/messages (Claude native format)
Client:   opencode/1.2.15 via @ai-sdk/anthropic (Claude native, NOT OpenAI format)
Model:    claude-opus(medium)
```

---

## Root Cause Chain

### 1. generateBillingHeader() — NON-DETERMINISTIC

**File**: `internal/runtime/executor/claude_executor.go:1252-1264`

```go
func generateBillingHeader(payload []byte) string {
    h := sha256.Sum256(payload)       // payload grows each turn → different hash
    cch := hex.EncodeToString(h[:])[:5]
    buildBytes := make([]byte, 2)
    _, _ = rand.Read(buildBytes)       // RANDOM EVERY REQUEST
    buildHash := hex.EncodeToString(buildBytes)[:3]
    return fmt.Sprintf("x-anthropic-billing-header: cc_version=2.1.63.%s; cc_entrypoint=cli; cch=%s;", buildHash, cch)
}
```

Two non-determinism sources:
- `rand.Read()` → random `buildHash` every request
- `sha256.Sum256(payload)` → `cch` changes because payload grows

### 2. checkSystemInstructionsWithMode() — RE-INJECTS EVERY TURN

**File**: `internal/runtime/executor/claude_executor.go:1271-1317`

The "already injected" guard checks `system.0.text` for "x-anthropic-billing-header:" prefix.
OpenCode sends its OWN fresh system prompt each turn (without proxy's injected blocks),
so the guard never triggers → proxy re-injects with NEW billing header every turn.

### 3. Signature Validation Fails

Claude's thinking signatures are HMAC-bound to:
- Thinking text (byte-perfect)
- Model ID
- **System prompt context** ← THIS CHANGES
- Session state

Turn 1 signature was computed with `system = [billing_v1, agent, user_prompt]`.
Turn 2 system prompt becomes `[billing_v2, agent, user_prompt]` (different billing header).
Signature verification fails → 400.

### 4. ensureCacheControl() — ADDITIONAL CONTEXT DRIFT

**File**: `internal/runtime/executor/claude_executor.go:1392-1406`

After cloaking, adds `cache_control: {"type": "ephemeral"}` to:
- Last system element
- Second-to-last user message's last content block
- Last tool definition

This also mutates the payload structure between turns, contributing to context drift.

---

## Request Pipeline (order matters)

```
1. sdktranslator.TranslateRequest()     — passthrough for claude→claude (OpenCode case)
2. thinking.ApplyThinking()             — thinking config, does NOT touch existing blocks
3. applyCloaking()                      — ⚠️ SYSTEM PROMPT CHANGES HERE
   ├─ checkSystemInstructionsWithMode() — billing header + agent ID injection
   ├─ injectFakeUserID()               — metadata.user_id (random if cacheUserID=false)
   └─ obfuscateSensitiveWords()         — zero-width space in sensitive words (text only, not thinking)
4. applyPayloadConfigWithRoot()         — config overlay
5. disableThinkingIfToolChoiceForced()  — tool_choice constraint
6. ensureCacheControl()                 — ⚠️ CACHE CONTROL MUTATIONS
7. enforceCacheControlLimit()           — max 4 breakpoints
8. normalizeCacheControlTTL()           — TTL ordering
9. applyClaudeToolPrefix()             — tool name prefix
```

---

## Why Intermittent

| Scenario | Result | Why |
|----------|--------|-----|
| First turn (no thinking blocks) | ✅ | No signature to validate |
| Turn 2+ without thinking in history | ✅ | No signature in messages |
| Turn 2+ with thinking, system changed | ❌ | **THIS BUG** |
| Claude prompt caching hit | ✅ sometimes | Cached context may match |
| Thinking disabled model | ✅ | No signatures at all |

---

## Fork Analysis

| Fork | Solves this? | Notes |
|------|-------------|-------|
| lemon (lemon07r) | ❌ | Fixes Antigravity translator path only |
| plusplus (KooshaPari) | ❌ | Behind upstream, still has enableThoughtTranslate bug |
| upstream (router-for-me) | ❌ | Same `rand.Read()` in generateBillingHeader |
| Our repo | Antigravity ✅, Direct Claude ❌ | antigravity_gemini_request.go already has isClaude branching |

Our repo is AHEAD of lemon for Antigravity path (we have the thinking strip patch from 9cc5d898).
The direct Claude API cloaking path is unsolved in ALL forks.

---

## Confirmed Fix Strategy (Oracle-validated)

### Strip historical thinking blocks BEFORE applyCloaking()

**Where**: `claude_executor.go` Execute() (line ~221) and ExecuteStream() (line ~397)

**Logic** (insert before `applyCloaking()` call):

```go
// Strip historical thinking blocks to prevent signature invalidation.
// Cloaking changes the system prompt (billing header + agent ID), which
// alters the security context that thinking signatures are bound to.
// Claude accepts conversations without prior thinking blocks.
body = stripHistoricalThinkingBlocks(body)
```

**Implementation sketch**:

```go
func stripHistoricalThinkingBlocks(payload []byte) []byte {
    messages := gjson.GetBytes(payload, "messages")
    if !messages.Exists() || !messages.IsArray() {
        return payload
    }

    // Walk messages in reverse to safely delete by index
    msgArray := messages.Array()
    for i := len(msgArray) - 1; i >= 0; i-- {
        msg := msgArray[i]
        if msg.Get("role").String() != "assistant" {
            continue
        }
        content := msg.Get("content")
        if !content.IsArray() {
            continue
        }

        // Walk content blocks in reverse, remove thinking blocks with signatures
        contentArray := content.Array()
        for j := len(contentArray) - 1; j >= 0; j-- {
            block := contentArray[j]
            blockType := block.Get("type").String()
            if blockType == "thinking" || blockType == "redacted_thinking" {
                sig := block.Get("signature").String()
                if sig != "" {
                    path := fmt.Sprintf("messages.%d.content.%d", i, j)
                    payload, _ = sjson.DeleteBytes(payload, path)
                }
            }
        }
    }
    return payload
}
```

### Why this is best

1. No session ID or billing header caching needed
2. Works for ALL clients (OpenCode, Roo-Code, curl, etc.)
3. Keeps cloaking fully functional
4. Minimal code change (~30 lines)
5. Covers ensureCacheControl() drift too (no signatures = nothing to validate)
6. Claude accepts conversations without prior thinking blocks

### Alternative: Redact instead of strip

Replace with `{"type":"thinking","thinking":"","signature":"redacted"}` instead of deleting.
Keeps message timeline shape. But stripping is simpler and confirmed safe.

---

## Open Investigation: How Real Claude Code Handles This

**TODO**: Research how the actual Claude Code CLI handles thinking signatures in multi-turn.
Does it:
- Send back thinking blocks unchanged? (likely — it controls the system prompt)
- Strip thinking blocks before sending?
- Use a deterministic billing header?
- Have some other mechanism?

The real Claude Code CLI doesn't have this problem because it controls the system prompt
end-to-end — it generated the billing header itself, so it's consistent across turns.
The proxy problem is that it injects into a system prompt it doesn't control.

---

## Key Files

```
internal/runtime/executor/claude_executor.go
  Line 221:  applyCloaking() call (Execute)
  Line 397:  applyCloaking() call (ExecuteStream)
  Line 1252: generateBillingHeader() — rand.Read + sha256
  Line 1271: checkSystemInstructionsWithMode() — system prompt injection
  Line 1228: injectFakeUserID() — metadata.user_id
  Line 1321: applyCloaking() — orchestrator function
  Line 1392: ensureCacheControl() — cache_control injection

internal/runtime/executor/cloak_utils.go
  Line 33:   shouldCloak() — auto/always/never
  Line 46:   isClaudeCodeClient() — UA check

internal/runtime/executor/cloak_obfuscate.go
  Line 86:   obfuscateSensitiveWords() — only touches type:"text", NOT thinking blocks ✅

internal/cache/signature_cache.go
  Full file: Signature caching for Antigravity path (already working)

internal/translator/antigravity/gemini/antigravity_gemini_request.go
  Line 185:  isClaude thinking strip (already fixed for Antigravity)

internal/translator/antigravity/claude/antigravity_claude_request.go
  Line 95:   Thinking block handling with signature cache (already fixed)
```

---

## Evidence Sources

- **4 explore agents**: Traced applyCloaking chain, conductor routing, thinking block flow, cloaking code
- **1 librarian agent**: Confirmed signatures bound to system prompt (Anthropic docs, Bedrock docs, GitHub issues)
- **1 oracle consultation**: Validated root cause, recommended strip/redact approach
- **2 fork repositories**: lemon07r + KooshaPari — neither solves direct Claude path
- **Debug logs**: 400 errors at messages.3.content.0, ~425-514ms response times

---

## Fix Applied (v2 — Thinking Block Stripping)

> **Date fixed**: 2026-03-01 (v1: deterministic headers), 2026-03-02 (v2: proxy-side strip)  
> **Status**: **RESOLVED**

The initial v1 fix (deterministic billing header + `context_management` injection + thinking-safe cache control) was insufficient. `context_management` with `clear_thinking_20251015` is processed server-side AFTER request validation — so the API still rejects invalid signatures before the directive takes effect.

### v2: Proxy-side thinking block stripping

`stripThinkingBlocks()` removes all `thinking` and `redacted_thinking` content blocks from assistant messages **before** `applyCloaking()` runs. Per Anthropic docs, prior-turn thinking blocks are optional — Claude accepts conversations without them.

This eliminates signature validation entirely:
- No signatures in the request → no validation → no 400 errors
- Works regardless of what cloaking does to system prompts
- Works for ALL clients (OpenCode, Roo-Code, curl, etc.)
- Empty assistant messages (thinking-only) are removed entirely

### v1 fixes retained

1. **Deterministic billing header** — `generateBillingHeader()` remains deterministic for cache hit stability
2. **Thinking-safe cache control** — `injectMessagesCacheControl()` still guards thinking blocks as a defense-in-depth measure
3. **`injectContextManagementIfThinking()` removed** — no longer needed; proxy-side strip handles the problem upstream
