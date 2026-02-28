# CLIProxyAPI Plus (anilcancakir fork)

English | [Chinese](README_CN.md)

This is an enhanced fork of [CLIProxyAPIPlus](https://github.com/router-for-me/CLIProxyAPIPlus), merging improvements from [lemon07r](https://github.com/lemon07r/CLIProxyAPIPlus) and [KooshaPari](https://github.com/KooshaPari/cliproxyapi-plusplus) forks on top of the mainline project.

The fork stays in sync with upstream via automated hourly sync and multi-arch Docker builds.

> [!NOTE]
> For a detailed breakdown of every fork-specific feature, see [FORK_EXTRA.md](FORK_EXTRA.md).

## What's Different

### Copilot — Claude & GPT-5 Routing

Tri-state endpoint routing enables Claude and GPT-5 models through GitHub Copilot:

- **Claude** (Sonnet, Opus) → `/v1/messages` with native Anthropic format
- **GPT-5 / Codex** → `/responses` with Responses API format
- **Legacy** (GPT-4o, etc.) → `/chat/completions` with OpenAI format

Headers match the official VS Code agent (`X-Initiator: agent`, dynamic session/machine IDs) for unlimited premium access. Thinking budgets are auto-configured for Sonnet 3.7+ models.

### Antigravity — Anti-Fingerprinting

- **Dynamic version fetching** from the auto-updater API (12h cache)
- **Per-account User-Agent** rotation based on auth ID hash
- **Version downgrade protection** per account
- **Session ID hardening** with auth-salted format

### Translator Fixes

- Thinking signature validation — invalid blocks silently dropped instead of 400 errors
- Consecutive same-role turn merging for Gemini
- Streaming tool call deltas in Claude-to-OpenAI translation
- Assistant prefill handling for Gemini models

### SDK Enhancements

- **Sticky Session Routing** — `X-Session-Key` header pins users to the same credential
- **Fallback Models** — automatic model degradation when primary is exhausted
- **Claude Request Sanitization** — strips placeholder fields from tool schemas
- **OpenAI Images API** — cross-provider image generation (DALL-E format → Gemini Imagen)
- **Extended Config Types** — SDK consumable as a Go library
- **OAuth Provider Priority** — YAML config controls provider preference; JSON `"priority"` field per-account overrides

### Additional Providers

- **Kilo AI** (OpenRouter) — dynamic model discovery, device flow auth, dedicated executor
- **Kiro Web Search** — MCP-based web search for AWS CodeWhisperer
- **Smart Routing** — `POST /v1/routing/select` for intent-based model selection

### CLI & Infrastructure

- **cliproxyctl** — CLI tool for setup, login, and diagnostics (`--json` output)
- **CI/CD** — hourly upstream sync + multi-arch Docker build to DockerHub
- **Dockerfile patches** — `patches/*.patch` applied during build for clean fork maintenance

## Supported Providers

| Provider | Auth Method | Features |
|:---------|:-----------|:---------|
| GitHub Copilot | OAuth | Claude, GPT-5, Codex, Legacy models |
| Antigravity | Token | Anti-fingerprinting, dynamic versioning, priority |
| Kiro (AWS) | OAuth | Web search, CodeWhisperer, priority |
| Kilo AI | Device Flow | OpenRouter models, dynamic discovery |
| Claude | API Key | Request sanitization, priority |
| Gemini / Vertex | API Key | Turn merging, image generation |
| Codex | WebSocket | Auto executor registration |
| OpenAI Compat | API Key | DALL-E / Imagen images |

## Quick Deployment with Docker

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

- [lemon07r/CLIProxyAPIPlus](https://github.com/lemon07r/CLIProxyAPIPlus) — Copilot Claude routing, anti-fingerprinting, translator fixes
- [KooshaPari/cliproxyapi-plusplus](https://github.com/KooshaPari/cliproxyapi-plusplus) — SDK enhancements, sticky routing, Kilo provider, CLI tooling
- [em4go](https://github.com/em4go/CLIProxyAPI/tree/feature/github-copilot-auth) — Original GitHub Copilot OAuth
- [fuko2935](https://github.com/fuko2935/CLIProxyAPI/tree/feature/kiro-integration), [Ravens2121](https://github.com/Ravens2121/CLIProxyAPIPlus/) — Kiro integration

## Contributing

This project only accepts pull requests that relate to third-party provider support. Any pull requests unrelated to third-party provider support will be rejected.

If you need to submit any non-third-party provider changes, please open them against the [mainline](https://github.com/router-for-me/CLIProxyAPI) repository.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
