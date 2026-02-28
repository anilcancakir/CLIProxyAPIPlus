# Port Plusplus SDK Features — Claude/Anthropic/OpenAI Focus

## TL;DR

> **Quick Summary**: Port remaining SDK-level improvements from the KooshaPari/cliproxyapi-plusplus fork to our fork (anilcancakir/CLIProxyAPIPlus), focusing on Claude Code, Anthropic protocol, Antigravity, and OpenAI enhancements. All changes are confined to `sdk/` and `test/` — zero modifications to `internal/`.
> 
> **Deliverables**:
> - Claude request sanitization (tool input schema cleanup)
> - Sticky session routing (StickyRoundRobinSelector)
> - Conductor fallback models logic with retryable errors
> - X-Session-Key header forwarding in handlers
> - Extended SDK config type exports
> - Updated service executor registration with CodexAutoExecutor
> - Enhanced error types with Retryable + HTTPStatus
> - Ported + adapted test suites
> 
> **Estimated Effort**: Medium
> **Parallel Execution**: YES — 5 waves
> **Critical Path**: Task 1 → Task 4 → Task 5 → Task 6 → Task 7

---

## Context

### Original Request
User wants to ensure their fork (`anilcancakir/CLIProxyAPIPlus`) has all relevant features from the `cliproxyapi-plusplus` fork, focusing on Claude Code, Anthropic protocol, Antigravity, and OpenAI improvements. The fork must remain sync-able with the upstream `router-for-me/CLIProxyAPIPlus`.

### Interview Summary
**Key Discussions**:
- `internal/` directory is 99.5% birebir identical — only 2 generated files missing (plusplus-specific codegen)
- Our fork has 5 EXTRA test files vs plusplus
- All functional code differences are import paths + formatting (our fork has better docblocks)
- SDK layer has significant feature gaps — conductor fallbacks, sticky routing, config exports, request sanitization

**Research Findings**:
- `NewClaudeAuth` constructor signature is IDENTICAL in both forks (`cfg *config.Config`) — plusplus `sdk/auth/claude.go` passes `nil` as second arg but that's the SDK wrapper, not the actual constructor
- `CodexAutoExecutor`, `NewOpenAICompatExecutor` both exist in our `internal/runtime/executor/`
- `RoutingConfig`, `CloakConfig`, `GeminiModel`, `QuotaExceeded`, `PprofConfig`, `AmpModelMapping` ALL exist in our `internal/config/config.go`
- `ProviderSpec`, `CursorKey`, `MiniMaxKey`, `DeepSeekKey`, etc. do NOT exist in our `internal/config/` — these are plusplus codegen-only types
- Module path: `github.com/router-for-me/CLIProxyAPI/v6` (correct, upstream-compatible)

### Metis Review
**Identified Gaps** (addressed):
- Risk of overwriting `Cancel context.CancelFunc` in StreamResult → Added explicit preservation guardrail
- Constructor signature mismatch risk → Verified: signatures are identical; plusplus SDK wrapper had extra nil arg
- Provider registration scope creep → Verified available executors in internal/; only register what exists
- `openai_images_handlers.go` depends on `pkg/llmproxy/` types → Will rewrite imports to `internal/`

---

## Work Objectives

### Core Objective
Bring the SDK layer of our fork to feature parity with plusplus for Claude/Anthropic/OpenAI concerns, while preserving upstream compatibility and our fork's improvements.

### Concrete Deliverables
- `sdk/api/handlers/claude/request_sanitize.go` + test — Claude tool schema sanitization
- `sdk/cliproxy/auth/selector.go` updated — StickyRoundRobinSelector + session routing
- `sdk/cliproxy/auth/conductor.go` updated — fallback models, retryable errors, executeWithFallback
- `sdk/api/handlers/handlers.go` updated — X-Session-Key forwarding
- `sdk/config/config.go` updated — missing type exports
- `sdk/cliproxy/service.go` updated — CodexAutoExecutor registration, provider entries
- `sdk/cliproxy/executor/types.go` updated — preserve Cancel, add context import if missing
- `sdk/auth/*.go` updated — auth wrapper improvements
- Test files ported and adapted

### Definition of Done
- [x] `go build ./...` passes with zero errors
- [x] `go test ./...` passes with zero failures
- [x] `go vet ./...` reports zero issues
- [x] All ported features compile and integrate correctly
- [x] No modifications to `internal/` directory

### Must Have
- Claude request sanitization (tool input schema cleanup)
- Sticky session routing via X-Session-Key
- Conductor fallback model chain
- Retryable + HTTPStatus on auth errors
- Extended config type exports for SDK consumers
- CodexAutoExecutor registration pattern
- All existing tests continue passing

### Must NOT Have (Guardrails)
- NO modifications to any file in `internal/` directory — SDK changes only
- NO importing from `pkg/llmproxy/` — all imports must use `internal/` paths
- NO deleting `Cancel context.CancelFunc` from StreamResult — this is our improvement
- NO porting plusplus codegen types (ProviderSpec, CursorKey, MiniMaxKey, etc.) — they don't exist in our internal/config
- NO registering providers that don't have executors in our `internal/runtime/executor/`
- NO changing module path — must stay `github.com/router-for-me/CLIProxyAPI/v6`
- NO over-commenting or excessive docblocks beyond what's necessary
- NO changing existing test expectations unless the test is for a feature being modified

---

## Verification Strategy

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES
- **Automated tests**: Tests-after (verify existing pass, port new tests)
- **Framework**: Go standard `testing` package
- **Command**: `go test ./...`

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Build verification**: `go build ./...`
- **Test verification**: `go test ./sdk/... -v` and `go test ./test/... -v`
- **Lint verification**: `go vet ./...`

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately — standalone additions):
├── Task 1: Export missing config types in sdk/config [quick]
├── Task 2: Port Claude request sanitization [quick]
└── Task 3: Port SDK handler/auth test files (standalone tests) [quick]

Wave 2 (After Wave 1 — routing core):
├── Task 4: Port StickyRoundRobinSelector + conductor fallbacks [deep]
└── Task 5: Port X-Session-Key handler forwarding + OpenAI images handler [unspecified-high]

Wave 3 (After Wave 2 — service integration):
└── Task 6: Update service.go executor registration + auth wrappers [unspecified-high]

Wave 4 (After Wave 3 — integration tests):
└── Task 7: Port e2e and integration tests [unspecified-high]

Wave FINAL (After ALL tasks — independent review, 4 parallel):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 1 → Task 4 → Task 6 → Task 7 → F1-F4
Parallel Speedup: ~40% faster than sequential
Max Concurrent: 3 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks | Wave |
|------|-----------|--------|------|
| 1 | — | 4, 5, 6 | 1 |
| 2 | — | 5 | 1 |
| 3 | — | 7 | 1 |
| 4 | 1 | 6 | 2 |
| 5 | 1, 2 | 6 | 2 |
| 6 | 4, 5 | 7 | 3 |
| 7 | 6, 3 | — | 4 |

### Agent Dispatch Summary

- **Wave 1**: **3** — T1 → `quick`, T2 → `quick`, T3 → `quick`
- **Wave 2**: **2** — T4 → `deep`, T5 → `unspecified-high`
- **Wave 3**: **1** — T6 → `unspecified-high`
- **Wave 4**: **1** — T7 → `unspecified-high`
- **FINAL**: **4** — F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

- [x] 1. Export Missing Config Types in sdk/config

  **What to do**:
  - Open `sdk/config/config.go` and add type aliases for all types that exist in `internal/config/config.go` but are missing from the SDK export layer
  - Add these missing exports: `RoutingConfig`, `AmpModelMapping`, `QuotaExceeded`, `PprofConfig`, `CloakConfig`, `GeminiModel`, `ClaudeModel`, `CodexModel`, `KiroKey`, `KiroFingerprintConfig`, `ClaudeHeaderDefaults`, `AmpUpstreamAPIKeyEntry`
  - Keep `internalconfig` alias naming (our fork's convention)
  - Do NOT add types that don't exist in `internal/config/`: `ProviderSpec`, `CursorKey`, `MiniMaxKey`, `DeepSeekKey`, `GroqKey`, `MistralKey`, `SiliconFlowKey`, `OpenRouterKey`, `TogetherKey`, `FireworksKey`, `NovitaKey`, `OAICompatProviderConfig`, `GeneratedConfig`
  - Verify every type you add actually exists: `grep 'type TypeName ' internal/config/config.go`

  **Must NOT do**:
  - Add type aliases for types that don't exist in `internal/config/`
  - Change existing type alias names or ordering
  - Modify any file in `internal/`

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3)
  - **Blocks**: Tasks 4, 5, 6
  - **Blocked By**: None

  **References**:
  - `sdk/config/config.go` — Current SDK config exports (the file to modify)
  - `internal/config/config.go` — Source of truth for all available config types. Lines 27-560 define all exportable types.
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/config/config.go` — Plusplus version showing which types they export (reference for what's missing)

  **Acceptance Criteria**:
  - [ ] `go build ./sdk/config/...` passes
  - [ ] `go build ./...` passes
  - [ ] All new type aliases resolve to real types in `internal/config/`

  **QA Scenarios:**
  ```
  Scenario: All exported types compile
    Tool: Bash
    Steps:
      1. Run: go build ./sdk/config/...
      2. Assert: exit code 0
    Expected Result: Zero build errors
    Evidence: .sisyphus/evidence/task-1-config-build.txt

  Scenario: No reference to nonexistent types
    Tool: Bash
    Steps:
      1. Run: go vet ./sdk/config/...
      2. Assert: exit code 0
    Expected Result: Zero vet errors
    Evidence: .sisyphus/evidence/task-1-config-vet.txt
  ```

  **Commit**: YES
  - Message: `feat(sdk): export missing config types for SDK consumers`
  - Files: `sdk/config/config.go`
  - Pre-commit: `go build ./sdk/config/...`

- [x] 2. Port Claude Request Sanitization

  **What to do**:
  - Copy `request_sanitize.go` from plusplus (`/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/claude/request_sanitize.go`)
  - Copy `request_sanitize_test.go` from plusplus (`/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/claude/request_sanitize_test.go`)
  - This file has NO imports from `pkg/llmproxy/` — it only uses `gjson`/`sjson` from tidwall. Should port cleanly.
  - Verify the package declaration is `package claude` (matching existing `sdk/api/handlers/claude/code_handlers.go`)
  - Verify `gjson` and `sjson` are already in `go.mod` (they should be — used elsewhere)

  **Must NOT do**:
  - Modify any existing file in `sdk/api/handlers/claude/`
  - Add any imports to `pkg/llmproxy/`

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3)
  - **Blocks**: Task 5
  - **Blocked By**: None

  **References**:
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/claude/request_sanitize.go` — Source file to port (137 lines). Uses gjson/sjson for JSON manipulation of Claude tool input schemas.
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/claude/request_sanitize_test.go` — Test file to port (150 lines). Table-driven tests for sanitization.
  - `sdk/api/handlers/claude/code_handlers.go` — Existing file in same package. Verify package name consistency.

  **Acceptance Criteria**:
  - [ ] `go build ./sdk/api/handlers/claude/...` passes
  - [ ] `go test ./sdk/api/handlers/claude/...` passes
  - [ ] No `pkg/llmproxy` imports in the new files

  **QA Scenarios:**
  ```
  Scenario: Claude sanitization builds and tests pass
    Tool: Bash
    Steps:
      1. Run: go build ./sdk/api/handlers/claude/...
      2. Assert: exit code 0
      3. Run: go test ./sdk/api/handlers/claude/... -v
      4. Assert: exit code 0, all tests PASS
    Expected Result: Build clean, all sanitization tests pass
    Evidence: .sisyphus/evidence/task-2-claude-sanitize-test.txt
  ```

  **Commit**: YES
  - Message: `feat(sdk): add Claude request sanitization for tool input schemas`
  - Files: `sdk/api/handlers/claude/request_sanitize.go`, `sdk/api/handlers/claude/request_sanitize_test.go`
  - Pre-commit: `go test ./sdk/api/handlers/claude/...`

- [x] 3. Port Missing SDK Test Files

  **What to do**:
  - Port these test files from plusplus, translating `pkg/llmproxy/` imports to `internal/`:
    - `sdk/access/manager_test.go` (86 lines)
    - `sdk/api/handlers/handlers_append_response_test.go` (27 lines)
    - `sdk/api/handlers/handlers_build_error_response_test.go` (35 lines)
    - `sdk/api/handlers/handlers_metadata_test.go` (85 lines)
    - `sdk/config/config_test.go` (41 lines)
  - For each file: read plusplus version, replace ALL `kooshapari/cliproxyapi-plusplus` with `router-for-me/CLIProxyAPI`, replace ALL `pkg/llmproxy/` path segments with `internal/`
  - Verify each test compiles and passes after porting
  - If a test references a type/function that doesn't exist in our fork, SKIP that specific test case and add a comment `// Skipped: type X not available in this fork`

  **Must NOT do**:
  - Modify existing test files
  - Add tests that import from `pkg/llmproxy/`
  - Create new production code to make tests pass

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2)
  - **Blocks**: Task 7
  - **Blocked By**: None

  **References**:
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/access/manager_test.go` — Test for access manager (86 lines)
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/handlers_append_response_test.go` — Handler append response test (27 lines)
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/handlers_build_error_response_test.go` — Error response builder test (35 lines)
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/handlers_metadata_test.go` — Handler metadata test (85 lines)
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/config/config_test.go` — Config test (41 lines)
  - `sdk/access/manager.go` — Existing file to verify test compatibility

  **Acceptance Criteria**:
  - [ ] `go test ./sdk/access/...` passes
  - [ ] `go test ./sdk/api/handlers/...` passes
  - [ ] `go test ./sdk/config/...` passes
  - [ ] No `pkg/llmproxy` imports in any ported test file

  **QA Scenarios:**
  ```
  Scenario: All ported SDK tests pass
    Tool: Bash
    Steps:
      1. Run: go test ./sdk/... -v 2>&1
      2. Assert: exit code 0
    Expected Result: All test packages pass
    Evidence: .sisyphus/evidence/task-3-sdk-tests.txt
  ```

  **Commit**: YES
  - Message: `test(sdk): port missing SDK test files from plusplus fork`
  - Files: `sdk/access/manager_test.go`, `sdk/api/handlers/handlers_*_test.go`, `sdk/config/config_test.go`
  - Pre-commit: `go test ./sdk/...`

- [x] 4. Port StickyRoundRobinSelector + Conductor Fallbacks + Retryable Errors

  **What to do**:
  This is the CORE routing improvement from plusplus. Three files to update:

  **A. `sdk/cliproxy/auth/types.go`** — Add `Retryable` and `HTTPStatus` fields to `Error` struct:
  - Read plusplus `sdk/cliproxy/auth/types.go` and compare with ours
  - Add `Retryable bool` and `HTTPStatus int` fields to the `Error` struct if missing
  - These fields are used by the conductor's `noAuthAvailableError()` helper

  **B. `sdk/cliproxy/auth/selector.go`** — Add StickyRoundRobinSelector:
  - Read plusplus `sdk/cliproxy/auth/selector.go`
  - Add the `StickyRoundRobinSelector` struct and its `Pick()` method (session-key-based routing)
  - Add the `sessionKeyMetadataKey` constant
  - Update the `noAuthAvailableError` inline error to include `Retryable: true` and `HTTPStatus: http.StatusServiceUnavailable`
  - Translate any `pkg/llmproxy/` imports to `internal/`
  - Keep all existing selector types (`FillFirstSelector`, `RoundRobinSelector`) untouched

  **C. `sdk/cliproxy/auth/conductor.go`** — Add fallback logic:
  - Add `noAuthAvailableError()` helper function
  - Refactor the existing execute methods to use `executeWithFallback()` pattern from plusplus
  - Add `fallback_models` context key support
  - Update error returns to use `noAuthAvailableError()` instead of inline `&Error{...}`
  - CRITICAL: Do NOT break the existing `executeMixedOnce` if it exists — carefully merge the fallback logic
  - Translate any `pkg/llmproxy/` imports to `internal/`

  **Must NOT do**:
  - Remove existing selector types
  - Change the selector interface signature
  - Modify `internal/` files
  - Remove `Cancel context.CancelFunc` from StreamResult (different file, but be aware)

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Task 5)
  - **Blocks**: Task 6
  - **Blocked By**: Task 1

  **References**:
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/cliproxy/auth/selector.go` — Plusplus selector with StickyRoundRobinSelector. Key additions: `StickyRoundRobinSelector` struct (lines 32-42), `Pick()` method (lines 379-458), `sessionKeyMetadataKey` constant.
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/cliproxy/auth/conductor.go` — Plusplus conductor with fallback logic. Key additions: `noAuthAvailableError()` (lines 86-94), `executeWithFallback()` (lines 603+), fallback_models context (lines 615-648).
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/cliproxy/auth/types.go` — Error type with Retryable and HTTPStatus fields.
  - `sdk/cliproxy/auth/selector.go` — Our current selector (to merge into, not overwrite)
  - `sdk/cliproxy/auth/conductor.go` — Our current conductor (to merge into, not overwrite)
  - `sdk/cliproxy/auth/types.go` — Our current types

  **Acceptance Criteria**:
  - [ ] `go build ./sdk/cliproxy/auth/...` passes
  - [ ] `go test ./sdk/cliproxy/auth/...` passes
  - [ ] `StickyRoundRobinSelector` type is accessible
  - [ ] `noAuthAvailableError()` helper exists in conductor
  - [ ] `Error` struct has `Retryable` and `HTTPStatus` fields

  **QA Scenarios:**
  ```
  Scenario: Routing core builds successfully
    Tool: Bash
    Steps:
      1. Run: go build ./sdk/cliproxy/auth/...
      2. Assert: exit code 0
      3. Run: go test ./sdk/cliproxy/auth/... -v
      4. Assert: exit code 0
    Expected Result: Zero build errors, all tests pass
    Evidence: .sisyphus/evidence/task-4-routing-build.txt

  Scenario: Sticky selector type is defined
    Tool: Bash
    Steps:
      1. Run: grep -n 'StickyRoundRobinSelector' sdk/cliproxy/auth/selector.go
      2. Assert: at least one match found
      3. Run: grep -n 'noAuthAvailableError' sdk/cliproxy/auth/conductor.go
      4. Assert: at least one match found
    Expected Result: Both new constructs exist
    Evidence: .sisyphus/evidence/task-4-routing-verify.txt
  ```

  **Commit**: YES
  - Message: `feat(sdk): add sticky session routing and conductor fallback models`
  - Files: `sdk/cliproxy/auth/selector.go`, `sdk/cliproxy/auth/conductor.go`, `sdk/cliproxy/auth/types.go`
  - Pre-commit: `go build ./sdk/cliproxy/auth/...`

- [x] 5. Port X-Session-Key Handler Forwarding + OpenAI Images Handler

  **What to do**:

  **A. `sdk/api/handlers/handlers.go`** — Add X-Session-Key forwarding:
  - Read plusplus `sdk/api/handlers/handlers.go` and identify the X-Session-Key block
  - Add session key extraction from gin context and forwarding to metadata
  - This is ~8 lines of code that reads `X-Session-Key` header and puts it in the `meta` map
  - Translate `pkg/llmproxy/` imports to `internal/`
  - Also check for `sdk/config` vs `internal/config` import difference and resolve

  **B. `sdk/api/handlers/openai/openai_images_handlers.go`** + test:
  - Read plusplus version. NOTE: It imports from `pkg/llmproxy/constant`, `pkg/llmproxy/interfaces`, `pkg/llmproxy/registry`
  - Translate these to: `internal/constant`, `internal/interfaces`, `internal/registry`
  - Copy and adapt the test file similarly
  - If any referenced type/function doesn't exist in `internal/`, add a TODO comment and adapt

  **Must NOT do**:
  - Break existing handler chain
  - Add `pkg/llmproxy/` imports
  - Modify `internal/` files

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Task 4)
  - **Blocks**: Task 6
  - **Blocked By**: Tasks 1, 2

  **References**:
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/handlers.go` — X-Session-Key forwarding logic (lines 202-209 in plusplus)
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/openai/openai_images_handlers.go` — OpenAI Images API handler (387 lines)
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/api/handlers/openai/openai_images_handlers_test.go` — Test (93 lines)
  - `sdk/api/handlers/handlers.go` — Our current handlers (merge target)
  - `sdk/api/handlers/openai/openai_handlers.go` — Existing OpenAI handler for pattern reference

  **Acceptance Criteria**:
  - [ ] `go build ./sdk/api/handlers/...` passes
  - [ ] `go test ./sdk/api/handlers/...` passes
  - [ ] X-Session-Key forwarding code is present in handlers.go
  - [ ] OpenAI images handler compiles

  **QA Scenarios:**
  ```
  Scenario: Handlers build with new features
    Tool: Bash
    Steps:
      1. Run: go build ./sdk/api/handlers/...
      2. Assert: exit code 0
      3. Run: go test ./sdk/api/handlers/... -v
      4. Assert: exit code 0
    Expected Result: All handler packages build and test cleanly
    Evidence: .sisyphus/evidence/task-5-handlers-build.txt

  Scenario: Session key forwarding present
    Tool: Bash
    Steps:
      1. Run: grep -n 'X-Session-Key\|session_key' sdk/api/handlers/handlers.go
      2. Assert: at least one match
    Expected Result: Session key forwarding code exists
    Evidence: .sisyphus/evidence/task-5-session-key-verify.txt
  ```

  **Commit**: YES
  - Message: `feat(sdk): add X-Session-Key forwarding and OpenAI images handler`
  - Files: `sdk/api/handlers/handlers.go`, `sdk/api/handlers/openai/openai_images_handlers.go`, `sdk/api/handlers/openai/openai_images_handlers_test.go`
  - Pre-commit: `go build ./sdk/api/handlers/...`

- [x] 6. Update Service Executor Registration + Auth Wrappers

  **What to do**:

  **A. `sdk/cliproxy/service.go`** — Update executor registration:
  - Key change: Use `CodexAutoExecutor` instead of `NewCodexExecutor` for codex provider
  - Add codex dedup logic: track `reboundCodex` flag to avoid registering codex executor multiple times
  - Add nil guard on `tokenResult`: `if tokenResult == nil { tokenResult = &TokenClientResult{} }`
  - Update `ensureExecutorsForAuthWithMode` to handle codex specially (check if already registered as CodexAutoExecutor)
  - For providers: ONLY register providers that have executors in `internal/runtime/executor/`. Available: `aistudio`, `antigravity`, `claude`, `codex`, `codex_websockets`, `gemini`, `gemini_cli`, `gemini_vertex`, `github_copilot`, `iflow`, `kilo`, `kimi`, `kiro`, `openai_compat`, `qwen`
  - Use `NewKiloExecutor` (not `NewOpenAICompatExecutor("kilo", cfg)`) for kilo provider
  - Translate `pkg/llmproxy/` imports to `internal/`

  **B. `sdk/auth/*.go`** — Auth wrapper alignment:
  - Check each auth wrapper file for import path differences
  - Our fork already uses correct `internal/` imports — only functional differences matter
  - `sdk/auth/claude.go`: Keep our constructor call `claude.NewClaudeAuth(cfg)` (verified identical signatures)
  - `sdk/auth/antigravity.go`, `sdk/auth/codex.go`: Keep our `val := N; return &val` pattern (more explicit than plusplus's approach)
  - Focus on any NEW functionality in auth wrappers, not style differences

  **Must NOT do**:
  - Register providers that don't have executor implementations
  - Remove existing provider registrations
  - Change `internal/` files
  - Remove `Cancel` from StreamResult (in executor/types.go)

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3 (sequential after Wave 2)
  - **Blocks**: Task 7
  - **Blocked By**: Tasks 4, 5

  **References**:
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/sdk/cliproxy/service.go` — Plusplus service with executor registration changes
  - `sdk/cliproxy/service.go` — Our current service (merge target, 550+ lines)
  - `internal/runtime/executor/codex_websockets_executor.go:1315-1340` — `CodexAutoExecutor` and `NewCodexAutoExecutor` definitions
  - `internal/runtime/executor/kilo_executor.go` — `NewKiloExecutor` definition
  - `internal/runtime/executor/openai_compat_executor.go:32` — `NewOpenAICompatExecutor` signature
  - `sdk/auth/claude.go`, `sdk/auth/antigravity.go`, `sdk/auth/codex.go` — Auth wrappers to review

  **Acceptance Criteria**:
  - [ ] `go build ./sdk/cliproxy/...` passes
  - [ ] `go build ./sdk/auth/...` passes
  - [ ] `go build ./...` passes (full build)
  - [ ] `go test ./sdk/...` passes
  - [ ] `CodexAutoExecutor` is used for codex provider registration

  **QA Scenarios:**
  ```
  Scenario: Full SDK builds with updated service
    Tool: Bash
    Steps:
      1. Run: go build ./...
      2. Assert: exit code 0
      3. Run: go test ./sdk/... -v
      4. Assert: exit code 0
    Expected Result: Everything builds, all tests pass
    Evidence: .sisyphus/evidence/task-6-service-build.txt

  Scenario: CodexAutoExecutor registration present
    Tool: Bash
    Steps:
      1. Run: grep -n 'CodexAutoExecutor\|NewCodexAutoExecutor' sdk/cliproxy/service.go
      2. Assert: at least one match
    Expected Result: CodexAutoExecutor is used
    Evidence: .sisyphus/evidence/task-6-codex-auto-verify.txt
  ```

  **Commit**: YES
  - Message: `feat(sdk): update service executor registration with CodexAutoExecutor`
  - Files: `sdk/cliproxy/service.go`, `sdk/auth/*.go`
  - Pre-commit: `go build ./...`

- [x] 7. Port E2E and Integration Tests

  **What to do**:
  - Port these test files from plusplus:
    - `test/e2e_test.go` (106 lines) — End-to-end test suite
    - `test/openai_websearch_translation_test.go` (313 lines) — OpenAI websearch translation tests
    - `test/roo_kilo_login_integration_test.go` (19 lines) — Roo/Kilo login integration test
  - Translate ALL `kooshapari/cliproxyapi-plusplus` to `router-for-me/CLIProxyAPI`
  - Translate ALL `pkg/llmproxy/` to `internal/`
  - If a test references types/functions unavailable in our fork, skip with comment
  - Run all existing + new tests to ensure nothing breaks

  **Must NOT do**:
  - Modify existing test files in `test/`
  - Add `pkg/llmproxy/` imports
  - Create stubs or mocks in `internal/` to make tests pass

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: [`anilcan-coding`]

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after Wave 3)
  - **Blocks**: None (final implementation task)
  - **Blocked By**: Tasks 3, 6

  **References**:
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/test/e2e_test.go` — E2E test (106 lines)
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/test/openai_websearch_translation_test.go` — Websearch translation test (313 lines)
  - `/Users/anilcan/Code/ai-help/repos/cliproxyforks/cliproxyapi-plusplus/test/roo_kilo_login_integration_test.go` — Integration test (19 lines)
  - `test/` — Our existing test dir (3 test files currently)

  **Acceptance Criteria**:
  - [ ] `go build ./test/...` passes
  - [ ] `go test ./test/... -v` passes
  - [ ] `go test ./... -count=1` passes (full regression)
  - [ ] No `pkg/llmproxy` imports

  **QA Scenarios:**
  ```
  Scenario: Full test suite passes
    Tool: Bash
    Steps:
      1. Run: go test ./... -count=1 2>&1
      2. Assert: exit code 0
      3. Assert: output contains no FAIL lines
    Expected Result: ALL packages pass, including new tests
    Evidence: .sisyphus/evidence/task-7-full-test.txt

  Scenario: No forbidden imports in test files
    Tool: Bash
    Steps:
      1. Run: grep -r 'pkg/llmproxy' test/
      2. Assert: exit code 1 (no matches)
    Expected Result: Zero plusplus-specific imports
    Evidence: .sisyphus/evidence/task-7-no-forbidden-imports.txt
  ```

  **Commit**: YES
  - Message: `test: port e2e and integration tests from plusplus fork`
  - Files: `test/e2e_test.go`, `test/openai_websearch_translation_test.go`, `test/roo_kilo_login_integration_test.go`
  - Pre-commit: `go test ./test/...`

---
## Final Verification Wave

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (`grep` for function/type, `go build`). For each "Must NOT Have": search codebase for forbidden patterns (`git diff` against `internal/`, check for `pkg/llmproxy` imports). Check evidence files exist in `.sisyphus/evidence/`. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `go build ./...` + `go vet ./...` + `go test ./...`. Review all changed SDK files for: type assertion safety, nil pointer risks, missing error handling, unused imports. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Vet [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Verify each ported feature compiles independently: `go build ./sdk/api/handlers/claude/...`, `go build ./sdk/cliproxy/auth/...`, `go build ./sdk/cliproxy/...`, `go build ./sdk/config/...`. Run all tests with `-race` flag: `go test -race ./...`. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Packages [N/N build] | Race Tests [N/N pass] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (`git diff`). Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check `internal/` has ZERO modifications (`git diff --name-only | grep internal/` must be empty). Detect cross-task contamination. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | internal/ [CLEAN/DIRTY] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

| Wave | Message | Files | Pre-commit |
|------|---------|-------|------------|
| 1 | `feat(sdk): export missing config types for SDK consumers` | `sdk/config/config.go` | `go build ./sdk/config/...` |
| 1 | `feat(sdk): add Claude request sanitization for tool input schemas` | `sdk/api/handlers/claude/request_sanitize.go`, `sdk/api/handlers/claude/request_sanitize_test.go` | `go test ./sdk/api/handlers/claude/...` |
| 1 | `test(sdk): port missing SDK test files from plusplus fork` | `sdk/access/manager_test.go`, `sdk/api/handlers/*_test.go`, `sdk/config/config_test.go` | `go test ./sdk/...` |
| 2 | `feat(sdk): add sticky session routing and conductor fallback models` | `sdk/cliproxy/auth/selector.go`, `sdk/cliproxy/auth/conductor.go`, `sdk/cliproxy/auth/types.go` | `go build ./sdk/cliproxy/auth/...` |
| 2 | `feat(sdk): add X-Session-Key forwarding and OpenAI images handler` | `sdk/api/handlers/handlers.go`, `sdk/api/handlers/openai/openai_images_handlers.go`, `sdk/api/handlers/openai/openai_images_handlers_test.go` | `go build ./sdk/api/handlers/...` |
| 3 | `feat(sdk): update service executor registration with CodexAutoExecutor` | `sdk/cliproxy/service.go`, `sdk/auth/*.go` | `go build ./sdk/...` |
| 4 | `test: port e2e and integration tests from plusplus fork` | `test/e2e_test.go`, `test/openai_websearch_translation_test.go`, `test/roo_kilo_login_integration_test.go` | `go test ./test/...` |

---

## Success Criteria

### Verification Commands
```bash
go build ./...                    # Expected: zero errors
go test ./... -count=1            # Expected: all PASS
go vet ./...                      # Expected: zero issues
git diff --name-only | grep -c 'internal/'  # Expected: 0 (no internal changes)
grep -r 'pkg/llmproxy' sdk/ test/ # Expected: no matches (no plusplus imports)
```

### Final Checklist
- [x] All "Must Have" features present and compilable
- [x] All "Must NOT Have" constraints verified (zero internal/ changes, no pkg/llmproxy imports)
- [x] All tests pass (existing + new)
- [x] Upstream sync maintained (no diverging internal/ changes)
- [x] `go build ./...` clean
- [x] `go vet ./...` clean
