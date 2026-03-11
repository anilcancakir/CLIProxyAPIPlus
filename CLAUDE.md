# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project

CLIProxyAPIPlus — multi-provider AI proxy that bridges 10+ providers (Claude, Antigravity/Gemini, OpenAI, GitHub Copilot, Kilo, Kiro, Codex, Qwen) behind unified OpenAI-compatible and Claude-native APIs. Fork of [router-for-me/CLIProxyAPIPlus](https://github.com/router-for-me/CLIProxyAPIPlus) with upstream sync every 12h. Fork-specific features documented in `FORK_EXTRA.md`.

## Commands

| Command | Description |
|---------|-------------|
| `go build ./cmd/server` | Build server binary |
| `go build ./cliproxyctl` | Build CLI control tool |
| `go test ./...` | Run all tests |
| `go vet ./...` | Static analysis |
| `go fmt ./...` | Format all Go files |
| `docker compose build` | Build Docker image (applies `patches/*.patch`) |

## Architecture

```
cmd/server/main.go              — API server entry point (:8317)
cliproxyctl/                     — CLI tool (setup, login, doctor)
sdk/                             — Public importable library
  auth/                          — Token interfaces + provider impls
  api/handlers/                  — HTTP route handlers (OpenAI, Claude, Gemini)
  cliproxy/                      — Core service, model registry, watcher
  config/                        — Re-exported config types
internal/                        — Private implementation
  runtime/executor/              — Provider dispatch (one executor per provider)
  translator/{proto}/{format}/   — Protocol adapters (Claude↔OpenAI↔Gemini)
  auth/{provider}/               — OAuth flows + token management
  config/                        — YAML parsing, hot-reload config struct
  watcher/                       — Config + auth file hot-reload (fsnotify)
  api/                           — Gin server, middleware, management endpoints
  tui/                           — Terminal dashboard UI
patches/                         — Git patches applied during Docker build
```

**Request flow:** HTTP → Gin middleware → sdk/api/handlers → internal/runtime/executor → translator → upstream API

## Key Files

- `config.example.yaml` — Full config reference (~200 lines, all provider options)
- `FORK_EXTRA.md` — Authoritative fork feature index with file locations and implementation details
- `internal/runtime/executor/claude_executor.go` — Cloaking, prompt caching, thinking strip, TLS bypass
- `internal/runtime/executor/antigravity_executor.go` — Anti-fingerprinting, quota tracking, rate limiting
- `internal/config/config.go` — Central config struct (all providers, routing, payload rules)
- `sdk/cliproxy/executor/types.go` — Executor interface, metadata key constants

## Code Style

- Executor interface: `Execute`, `ExecuteStream`, `Refresh`, `CountTokens`, `HttpRequest`
- Metadata keys via exported constants (`RequestedModelMetadataKey`, `PinnedAuthMetadataKey`) — never hardcode strings
- `StatusError` interface for HTTP-aware errors (`StatusCode() int`) — check with type assertion
- Config is hot-reload safe: always read from live `cfg` pointer at call time, never cache at init
- Provider constants defined in `internal/constant/constant.go`

## Testing

- `go test ./...` — all unit + integration tests (testify assertions)
- Table-driven tests with `[]struct{ input, want }` pattern
- E2E tests in `test/` directory (health checks, integration flows)
- New features require corresponding `*_test.go` files

## Gotchas

- **Thinking blocks:** Strip ALL `thinking`/`redacted_thinking` blocks BEFORE `applyCloaking()` — system prompt mutations invalidate signatures. Never add `cache_control` to thinking blocks.
- **Translator path guard:** `internal/translator/**` changes are blocked by CI (`pr-path-guard.yml`) — restricted to maintainers.
- **Copilot dual routing:** GPT-5/Codex → `/responses` (Responses API), all others → `/chat/completions` (OpenAI). Never mix formats on a single endpoint.
- **Antigravity identity:** Per-account User-Agent via auth ID hash. Session IDs salted with auth context (`-{uuid}:{model}:{project}:seed-{hex16}`). Rate limiter uses `accountID:modelName` keys for model-level isolation.
- **Quota threshold:** Pre-execution check compares `FiveHour.Utilization` against configured threshold — returns fast 429 before any HTTP work. Cold starts (no cached quota) pass through.
- **Beta header resilience:** Essential betas (`claude-code-*`, `context-management-*`, `interleaved-thinking-*`) are force-appended when missing from client headers.
- **Streaming:** Always call `StreamResult.Cancel()` when done — prevents goroutine leaks. Wrap channels in `StreamResult`, never return raw `chan`.
- **Build metadata:** `Version`, `Commit`, `BuildDate` injected via ldflags at link time (`cmd/server/main.go`).

## Skills & Extensions

- MCP: `context7` — live framework docs, version-aware library reference
