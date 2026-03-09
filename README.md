## What's Different

[![z.ai](https://assets.router-for.me/english-5-0.jpg)](https://z.ai/subscribe?ic=8JVLJQFSKB)

### Copilot — OpenCode Client Integration

Dual-state endpoint routing for GitHub Copilot models with **OpenCode-approved client** configuration:

- **GPT-5 / Codex** → `/responses` with Responses API format
- **All other models** (Claude, GPT-4o, etc.) → `/chat/completions` with OpenAI format

Claude models are routed through the standard OpenAI-format translation path — the proxy handles format conversion automatically. The proxy uses OpenCode's official GitHub Client ID (`Ov23li8tweQw6odWQebz`) and simplified scope (`read:user`) for approved client status.

**OpenCode Client Configuration:**
- Client ID: `Ov23li8tweQw6odWQebz` (OpenCode's OAuth client)
- User-Agent: `opencode/0.1.0` (OpenCode pattern)
- OpenAI-Intent: `conversation-edits`
- Dynamic X-Initiator header based on conversation role (agent/user)
- Removed VSCode-specific headers for cleaner OpenCode compatibility

### Antigravity — Anti-Fingerprinting

- **Dynamic version fetching** from the auto-updater API (12h cache)
- **Per-account User-Agent** rotation based on auth ID hash
- **Version downgrade protection** per account
- **Session ID hardening** with auth-salted format
- **Quota Tracking** — proactive quota awareness via management API, reason-based backoff, model-level rate limit isolation

### Claude Code — Cloaking & Prompt Caching

- **Request Cloaking** — any client (curl, OpenAI SDK, Roo-Code) is transparently disguised as Claude Code CLI v2.1.63 via deterministic billing header injection, agent system prompt, fake user_id, and header emulation
- **Cloak modes** — `auto` (default, cloak non-Claude clients), `always`, `never`; strict mode strips user system messages to enforce consistent identity
- **Automated Prompt Caching** — auto-injects `cache_control: {"type": "ephemeral"}` breakpoints (up to 4 per request) across tools, system, and messages layers for up to 90% cost reduction on repeated prompts; thinking/redacted_thinking blocks are excluded to preserve signature integrity
- **Thinking Signature Stability** — proxy-side stripping of historical thinking blocks before cloaking prevents `Invalid signature` 400 errors in multi-turn conversations; deterministic billing headers and thinking-safe cache control provide additional safety
- **1M Context Window** — `X-CPA-CLAUDE-1M` request header auto-injects the `context-1m-2025-08-07` beta for 1M token context support
- **Beta Header Resilience** — essential cloaking betas (`claude-code`, `oauth`, `interleaved-thinking`, `context-management`, `prompt-caching-scope`) are force-appended when clients send their own `Anthropic-Beta` header, preventing identity leaks
- **TLS Fingerprint Bypass** — `utls` with `tls.HelloChrome_Auto` fingerprint and HTTP/2 connection pooling matches the real Claude Code wire footprint
- **OAuth2 PKCE** — `cliproxyctl login --provider claude` runs a local callback flow; tokens stored as `claude-{email}.json` and auto-refreshed
- **Quota Threshold Fallback** — per-model 5-hour utilization thresholds trigger fast 429 errors before the API call, letting the conductor fall back to alternative providers (Antigravity, Copilot) via existing priority routing

### Model-Based System Prompt Injection

Inject custom system prompts per model alias with wildcard pattern matching:

```yaml
payload:
  system-prompts:
    - models:
        - name: "can"
          protocol: "openai"
      prompt: "You are Can, a helpful AI assistant."
      mode: prepend
```

- **Pattern matching**: `claude-*`, `*-opus-*`, `gpt-4*`, exact matches
- **Injection modes**: prepend (default), append, replace
- **Protocol support**: Claude (system array), OpenAI/Gemini (messages array)
- **Works with**: All providers including OpenAI-compatibility providers

### Translator Fixes

- Thinking signature validation — invalid blocks silently dropped instead of 400 errors
- Consecutive same-role turn merging for Gemini
- Streaming tool call deltas in Claude-to-OpenAI translation
- Assistant prefill handling for Gemini models
- `reasoning_text` → `thinking` block conversion for Copilot/Gemini models in OpenAI-to-Claude translation

### SDK Enhancements

- **Sticky Session Routing** — `X-Session-Key` header pins users to the same credential
- **Fallback Models** — automatic model degradation when primary is exhausted
- **Claude Request Sanitization** — strips placeholder fields from tool schemas
- **OpenAI Images API** — cross-provider image generation (DALL-E format → Gemini Imagen)
- **Extended Config Types** — SDK consumable as a Go library

### OAuth Provider Priority

User-controllable priority for OAuth providers and individual accounts:

- **Provider-level** — `oauth-provider-priority` map in `config.yaml` sets integer priority per provider
- **Account-level** — `"priority"` field in a JSON auth file overrides the provider config
- Resolution: JSON account priority > provider config > default 0
- Explicit `"priority": 0` in JSON overrides a non-zero provider config (zero is not "unset")
- Backward compatible — missing `oauth-provider-priority` in config works unchanged

### Additional Providers

- **Kilo AI** (OpenRouter) — dynamic model discovery, device flow auth, dedicated executor
- **Kiro Web Search** — MCP-based web search for AWS CodeWhisperer
- **Smart Routing** — `POST /v1/routing/select` for intent-based model selection

### CLI & Infrastructure

- **cliproxyctl** — CLI tool for setup, login, and diagnostics (`--json` output)
- **CI/CD** — 12-hour upstream sync + amd64 Docker build to DockerHub; upstream merge conflicts fail the build for manual resolution
- **Dockerfile patches** — `patches/*.patch` applied during build for clean fork maintenance

## Supported Providers

| Provider | Auth Method | Features |
|:---------|:-----------|:---------|
| GitHub Copilot | OAuth (OpenCode) | Claude, GPT-5, Codex, approved client status |
| Antigravity | Token | Anti-fingerprinting, dynamic versioning, quota tracking, priority |
| Kiro (AWS) | OAuth | Web search, CodeWhisperer, priority |
| Kilo AI | Device Flow | OpenRouter models, dynamic discovery |
| Claude | API Key / OAuth | Request cloaking, prompt caching, TLS bypass, priority |
| Gemini / Vertex | API Key | Turn merging, image generation |
| Codex | WebSocket | Auto executor registration |
| OpenAI Compat | API Key | System prompt injection, model aliases, DALL-E / Imagen images |

## Quick Deployment with Docker

Pre-built images are published to [Docker Hub](https://hub.docker.com/r/anilcancakir/cli-proxy-api-plus).
### One-Command Deployment

```bash
mkdir -p ~/cli-proxy && cd ~/cli-proxy

cat > docker-compose.yml << 'EOF'
services:
  cli-proxy-api:
    image: anilcancakir/cli-proxy-api-plus:latest
    container_name: cli-proxy-api-plus
    ports:
      - "8317:8317"
    volumes:
      - ./config.yaml:/CLIProxyAPI/config.yaml
      - ./auths:/root/.cli-proxy-api
      - ./logs:/CLIProxyAPI/logs
    restart: unless-stopped
EOF

curl -o config.yaml https://raw.githubusercontent.com/anilcancakir/CLIProxyAPIPlus/main/config.example.yaml

docker compose pull && docker compose up -d
```

### Configuration

Copy the example config and customize:

```bash
# Basic config
cp config.example.yaml config.yaml

# Extended config with Copilot/Antigravity aliases
cp config.example.custom.yaml config.yaml

# OpenCode/Roo-Code integration reference
cat example.opencode.json
```

### Update to Latest Version

```bash
cd ~/cli-proxy
docker compose pull && docker compose up -d
```

Auto-update is also available via the included `docker-auto-update.sh` and `setup-cron.sh` scripts.

## Kiro Authentication

Access the Kiro OAuth web interface at:

```
http://your-server:8317/v0/oauth/kiro
```

This provides a browser-based OAuth flow for Kiro (AWS CodeWhisperer) authentication with:
- AWS Builder ID login
- AWS Identity Center (IDC) login
- Token import from Kiro IDE

## cliproxyctl

The CLI control tool provides proxy management without editing YAML:

```bash
# Interactive setup wizard
cliproxyctl setup

# OAuth login (Gemini, Kiro)
cliproxyctl login --provider kiro

# Diagnostic check
cliproxyctl doctor
cliproxyctl doctor --json  # machine-readable output
```

## Attribution

This fork incorporates work from:

- [lemon07r/CLIProxyAPIPlus](https://github.com/lemon07r/CLIProxyAPIPlus) — Copilot GPT-5 routing, anti-fingerprinting, translator fixes
- [KooshaPari/cliproxyapi-plusplus](https://github.com/KooshaPari/cliproxyapi-plusplus) — SDK enhancements, sticky routing, Kilo provider, CLI tooling
- [em4go](https://github.com/em4go/CLIProxyAPI/tree/feature/github-copilot-auth) — Original GitHub Copilot OAuth
- [fuko2935](https://github.com/fuko2935/CLIProxyAPI/tree/feature/kiro-integration), [Ravens2121](https://github.com/Ravens2121/CLIProxyAPIPlus/) — Kiro integration

