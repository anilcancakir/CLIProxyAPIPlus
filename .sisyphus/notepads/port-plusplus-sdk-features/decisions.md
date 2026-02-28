# Decisions — port-plusplus-sdk-features

## Session Started: 2026-02-28T12:59:42.633Z

## Key Decisions from Plan
- Module path stays: `github.com/router-for-me/CLIProxyAPI/v6`
- NO modifications to `internal/` directory
- NO `pkg/llmproxy/` imports — use `internal/` instead
- NO porting plusplus codegen types (ProviderSpec, CursorKey, etc.)
- Keep `Cancel context.CancelFunc` in StreamResult
- Use `NewKiloExecutor` (not `NewOpenAICompatExecutor("kilo", cfg)`) for kilo
